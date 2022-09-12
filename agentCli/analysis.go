package agentCli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gorhill/cronexpr"
	"github.com/openark/golib/log"
	"github.com/openark/golib/sqlutils"
	"github.com/spf13/cobra"

	"op-agent/db"
)

func newAnalysis() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analysis <packages|jobs> [jobName] [onceJobVersion]",
		Short: "Analyze <packages|jobs> status",
		Long:  `Example:
				analysis packages
				analysis jobs [jobName] [onceJobVersion]
		`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				sourceKind string
				jobName string
				onceJobVersion string
			)

			if len(args) == 0 {
				log.Errorf("packages or jobs must be provided")
				return nil
			}
			sourceKind = args[0]
			if sourceKind == "packages" {
				checkPackages()
			} else if sourceKind == "jobs" {
				if len(args) > 1 {
					jobName = args[1]
				}
				if len(args) > 2 {
					onceJobVersion = args[2]
				}
				checkJobs(jobName, onceJobVersion)
			} else {
				log.Errorf("Just support jobs|packages param")
				return nil
			}
			return nil
		},
	}
	return cmd
}

func checkJobs(jobName string, version string) {
	var (
		dataLists [][]string
	)
	dataLists = [][]string{}
	jobSpecList := getJobSpec(jobName, version)
	if len(jobSpecList) == 0 {
		log.Infof("getJobSpec is null, the job might be deleted or disabled. Pls check.")
		return
	}
	for _, jobSpec := range jobSpecList {
		results := checkOneJob(jobSpec)
		dataLists = append(dataLists, results...)
	}
	table := TableWriter([]string{"JobName", "Host", "Message"}, dataLists)
	table.Render()

}

func checkOneJob(jobSpec *JobSpec) (results [][]string) {
	var (
		err error
		activeHosts []map[string]string
	)
	results = [][]string{}
	if activeHosts, err = GetAllActiveHosts(); err != nil {
		log.Errorf("GetAllActiveHosts err when check %s : %+v", jobSpec.JobName, err)
		return
	}
    for _, hostMap := range activeHosts {
			var (
				runSeconds int64
				lastRunSeconds int64
				findRowFlag bool
				outPut string
				outErr string
				result []string
				onceJobHasRun string
				status string
			)
			result = []string{}
			if jobSpec.CheckBlackIps(hostMap["ip"]) {
				continue
			}
			query := ""
			whereCondition := fmt.Sprintf("token='%s' and jobname='%s'", hostMap["token"], jobSpec.JobName)
			if jobSpec.OnceJob == 1 {
				if jobSpec.Version != "" {
					whereCondition = fmt.Sprintf("version='%s' and jobname='%s' and token='%s'", jobSpec.Version, jobSpec.JobName, hostMap["token"])
				}
				query = `select 
							timestampdiff(SECOND, starttime, endtime) as runseconds, 
							timestampdiff(SECOND, endtime, now()) as lastrunsconds,
							endtime, output, err, hasrun, status   
						from oncejobtask where %s order by version desc limit 1`
			} else {
				query = `select 
							timestampdiff(SECOND, starttime, endtime) as runseconds, 
							timestampdiff(SECOND, endtime, now()) as lastrunsconds,
							endtime, output, err  
						from joblogs where %s order by _timestamp desc limit 1`
			}
			query = fmt.Sprintf(query, whereCondition)
			db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
				runSeconds = m.GetInt64("runseconds")
				lastRunSeconds = m.GetInt64("lastrunsconds")
				outPut = m.GetString("output")
				outErr = m.GetString("err")
				if jobSpec.OnceJob == 1 {
					onceJobHasRun = m.GetString("hasrun")
					status = m.GetString("status")
				}
				findRowFlag = true
				return nil
			})

			if jobSpec.OnceJob == 0 {
				if findRowFlag {
					if outErr != "" {
						result = append(result, jobSpec.JobName)
						result = append(result, hostMap["ip"])
						result = append(result, outErr)
					} else {
						if jobSpec.CronExpr != "" {
							info := fmt.Sprintf(`Job %s has not run for a long time. LasteRunFromNowSeconds: %d, SchedulRunInterval: %d`,
								jobSpec.JobName, lastRunSeconds, jobSpec.RunIntervalSeconds)
							if runSeconds < jobSpec.RunIntervalSeconds {
								if lastRunSeconds > 2*jobSpec.RunIntervalSeconds {
									result = append(result, jobSpec.JobName)
									result = append(result, hostMap["ip"])
									result = append(result, info)
								}
							} else {
								if lastRunSeconds > runSeconds+jobSpec.RunIntervalSeconds {
									result = append(result, jobSpec.JobName)
									result = append(result, hostMap["ip"])
									result = append(result, info)
								}
							}
						}
					}
				} else {
					if jobSpec.CronExpr != "" {
						if jobSpec.AddFromNowSeconds > jobSpec.RunIntervalSeconds {
							info := fmt.Sprintf(`Job %s did not run. AddFromNowSeconds: %d, SchedulRunInterval: %d`,
								jobSpec.JobName, jobSpec.AddFromNowSeconds, jobSpec.RunIntervalSeconds)
							result = append(result, jobSpec.JobName)
							result = append(result, hostMap["ip"])
							result = append(result, info)
						}
					}
				}
			}


			if jobSpec.OnceJob == 1 {
				if !findRowFlag {
					continue
				}
				if onceJobHasRun == "1" {
					if outErr != "" {
						result = append(result, jobSpec.JobName)
						result = append(result, hostMap["ip"])
						result = append(result, outErr)
					}
				} else {
					if jobSpec.AddFromNowSeconds > 30 {
						info := ""
						if status != "1" {
							info = fmt.Sprintf(`onceJob %s did not run. AddFromNowSeconds: %d, SchedulRunInterval: %d`,
								jobSpec.JobName, jobSpec.AddFromNowSeconds, jobSpec.RunIntervalSeconds)
						} else if status == "1" {
							info = fmt.Sprintf("onceJob %s is running. AddFromNowSeconds: %d, SchedulRunInterval: %d",
								jobSpec.JobName, jobSpec.AddFromNowSeconds, jobSpec.RunIntervalSeconds)
						}
						result = append(result, jobSpec.JobName)
						result = append(result, hostMap["ip"])
						result = append(result, info)
					}
				}

			}

			if len(result) != 0 {
				results = append(results, result)
			}
	}
	return
}


func getJobSpec(jobName string, version string) []*JobSpec {
	var (
		queryJobSpec string
		jobSpecs []*JobSpec
	)
	jobSpecs = []*JobSpec{}
	queryJobSpec = `
			select 
					jobname, command,cronexpr, oncejob,whiteips, blackips, 
					timestampdiff(SECOND, _timestamp, now()) as add_seconds 
			from jobs
	`
	if jobName == "" {
		queryJobSpec =  queryJobSpec + " where enabled=1"
	} else {
		queryJobSpec = queryJobSpec + " where enabled=1 and jobname='%s'"
		queryJobSpec = fmt.Sprintf(queryJobSpec, jobName)
	}

	db.QueryDBRowsMap(queryJobSpec, func(m sqlutils.RowMap) error {
		var (
			err error
		)
		jobSpec := &JobSpec{}
		jobSpec.JobName = m.GetString("jobname")
		jobSpec.Command = m.GetString("command")
		jobSpec.CronExpr = m.GetString("cronexpr")
		jobSpec.OnceJob = m.GetUint("oncejob")
		jobSpec.WhiteIps = m.GetString("whiteips")
		jobSpec.BlackIps = m.GetString("blackips")
		jobSpec.Version = version
		jobSpec.AddFromNowSeconds = m.GetInt64("add_seconds")
		if m.GetUint("oncejob") != 1 && m.GetString("cronexpr") != "" {
			if jobSpec.Expr, err = cronexpr.Parse(jobSpec.CronExpr); err != nil {
				return err
			}
		}
		if err := jobSpec.CalculateRunIntervalSeconds(); err != nil {
			log.Errorf("Check CronExpr of RunIntervalSeconds %+v", jobSpec.JobName ,err)
		}
		jobSpecs = append(jobSpecs, jobSpec)
		return nil
	})
	return jobSpecs
}


func checkPackages() (dataLists [][]string){
	var (
		err error
		activeHosts []map[string]string
		allNewPackages []string
		notHaveNewPackageList []string
	)
	dataLists = [][]string{}
	allNewPackages = []string{}
	notHaveNewPackageList = []string{}

	activeHosts, err = GetAllActiveHosts()
	if err != nil {
		log.Errorf("GetAllActiveHosts err %+v", err)
		return
	}

	query := fmt.Sprintf(`
			select 
				concat(package_name, max(package_version)) as packageinfo  
			from  package_info group by package_name
		`)
	rowsMap, _ := db.QueryAll(query)
	for _, row := range rowsMap {
		allNewPackages = append(allNewPackages, row["packageinfo"])
	}
	sort.Strings(allNewPackages)

	for _, hostMap := range activeHosts {
		hostPackages := []string{}
		query := `select 
						concat(package_name, package_version) as packageinfo 
				  from agent_package_info where token='%s'`
		query = fmt.Sprintf(query, hostMap["token"])
		rowsMap, _ := db.QueryAll(query)
		for _, row := range rowsMap {
			hostPackages = append(hostPackages, row["packageinfo"])
		}
		sort.Strings(hostPackages)
		if strings.Join(hostPackages, ",") != strings.Join(allNewPackages, ",") {
			notHaveNewPackageList = append(notHaveNewPackageList, hostMap["ip"])
			notHaveNewPackageList = append(notHaveNewPackageList, "Package is not up to date")
		}
		dataLists = append(dataLists, notHaveNewPackageList)
		notHaveNewPackageList = []string{}
	}
	table := TableWriter([]string{"HostIP", "PackageStatus"}, dataLists)
	table.Render()
	return
}
