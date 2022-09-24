package agentCli

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/olekukonko/tablewriter"
	"github.com/openark/golib/log"
	"github.com/openark/golib/sqlutils"

	"op-agent/db"
)

type AgentNode struct {
	Hostname 			string
	IP 					string
	Token				string
	Port     			int
	Outdated			bool
	ActiveFlag			int
	Ctime				string
	LastSeenActive		string
}

type JobSpec struct {
	JobName string
	Command string
	CronExpr string
	Expr *cronexpr.Expression
	NextTimeList []time.Time
	RunIntervalSeconds int64
	AddFromNowSeconds int64
	OnceJob uint
	Version string
	WhiteIps string
	BlackIps string
}

func (jobSpec *JobSpec) CheckBlackIps(hosts string) bool {
	var (
		findInWhite bool
	)
	ipsList := strings.Split(hosts, ",")
	for _, ip := range ipsList {
		if strings.Contains(jobSpec.BlackIps, ip) {
			return true
		}
		if  jobSpec.WhiteIps != "" {
			if strings.Contains(jobSpec.WhiteIps, ip) {
				findInWhite = true
			}
		}
	}
	if jobSpec.WhiteIps != "" {
		if !findInWhite {
			return true
		}
	}
	return false
}

func (jobSpec *JobSpec) CalculateRunIntervalSeconds() error {
	var (
		err error
		now time.Time
	)
	if jobSpec.OnceJob == 1 || jobSpec.CronExpr == "" {
		return nil
	}
	now = time.Now()
	if jobSpec.Expr, err = cronexpr.Parse(jobSpec.CronExpr); err != nil {
		return err
	}
	jobSpec.NextTimeList = jobSpec.Expr.NextN(now,3)
	thirdTime := jobSpec.NextTimeList[2]
	secondTime := jobSpec.NextTimeList[1]
	jobSpec.RunIntervalSeconds = int64(thirdTime.Sub(secondTime).Seconds())
	return nil
}

func TableWriter(title []string, data [][]string ) *tablewriter.Table {
	var (
		table *tablewriter.Table
	)

	table = tablewriter.NewWriter(os.Stdout)
	table.SetHeader(title)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	table.AppendBulk(data) // Add Bulk Data
	return table
}

func GetAllActiveHosts() ([]map[string]string, error) {
	 var (
	 	err error
	 	dataMapLists []map[string]string
	 	results  []map[string]string
	 )
	 results = []map[string]string{}
	 /*query := `select *
			  from 
					node_health 
				where last_seen_active  between date_add(now(), interval - 10 minute) and now()`
	 err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		result := map[string]string{}
		result["hostname"] = m.GetString("hostname")
		result["ip"] = m.GetString("ip")
		result["token"] = m.GetString("token")
		result["port"] = m.GetString("http_port")
		result["active_flag"] = m.GetString("active_flag")
		result["last_seen_active"] = m.GetString("last_seen_active")
		results = append(results, result)
		return nil
	})*/
	//results, err = db.QueryAll(query)
	dataMapLists, err = TakeAgentsStatus("","")
	for _, data := range dataMapLists {
		if data["status"] == "Ready" {
			results = append(results, data)
		}
	}
	return results, err
}

func GetAllAgentHosts() ([]map[string]string, error) {
	var (
		results  []map[string]string
		err error
	)
	results = []map[string]string{}
	query := `select * from node_health order by last_seen_active desc`
	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		result := map[string]string{}
		result["hostname"] = m.GetString("hostname")
		result["ip"] = m.GetString("ip")
		result["token"] = m.GetString("token")
		result["port"] = m.GetString("http_port")
		result["active_flag"] = m.GetString("active_flag")
		result["last_seen_active"] = m.GetString("last_seen_active")
		results = append(results, result)
		return nil
	})
	return results, err
}


func GetOutDatedAgentHosts() ([]map[string]string, error) {
	var (
		results  []map[string]string
		err error
	)
	results = []map[string]string{}
	query := `select * from node_health where last_seen_active  not between date_add(now(), interval - 2 minute) and now()  order by last_seen_active desc`
	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		result := map[string]string{}
		result["hostname"] = m.GetString("hostname")
		result["ip"] = m.GetString("ip")
		result["token"] = m.GetString("token")
		result["port"] = m.GetString("http_port")
		result["active_flag"] = m.GetString("active_flag")
		result["last_seen_active"] = m.GetString("last_seen_active")
		results = append(results, result)
		return nil
	})
	return results, err
}

func PurgeOutDatedAgentsHosts() error {
	delete := "delete from node_health where last_seen_active not between date_add(now(), interval - 1440 minute) and now()" //1440 minute = 24 hour
	_, err := db.ExecDb(delete)
	return err
}

func GetNeedDownloadPluginAgents() ([]map[string]string, error) {
	var (
		results  []map[string]string
		err error
	)
	results = []map[string]string{}
	query := "select t.hostname, t.token, t.agent_ips, h.http_port port from agent_package_task t left join node_health h on t.token=h.token  where t.status='1' and h.http_port !=''"
	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		result := map[string]string{}
		result["hostname"] = m.GetString("hostname")
		result["ip"] = m.GetString("agent_ips")
		result["token"] = m.GetString("token")
		result["port"] = m.GetString("port")
		results = append(results, result)
		return nil
	})
	return results, err
}

func GetAllActiveAgentsWhenAddNewJob() ([]map[string]string, error) {
	var (
		results  []map[string]string
		newJobFlag bool
		err error
	)
	results = []map[string]string{}
	query := "select count(1) count from jobs where _timestamp  between date_add(now(), interval - 1 minute) and now()"
	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		if m.GetInt("count") > 1 {
			newJobFlag = true
		}
		return nil
	})
	if newJobFlag {
		results, err = GetAllActiveHosts()
	}
	return results, err
}

func GetAllJobs(token string) ([]map[string]string, error) {
	var (
		query string
		err error
		results []map[string]string
	)
	results = make([]map[string]string,0)
	query = fmt.Sprintf("select * from jobs where enabled = 1")

	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		resultMap := map[string]string{}
		onceJob := m.GetString("oncejob")
		jobname := m.GetString("jobname")
		if onceJob == "1" {
			resultMap["oncejobversion"] = getOnceJobVersion(token, jobname)
		} else {
			resultMap["oncejobversion"] = ""
		}
		resultMap["oncejob"] = onceJob
		resultMap["jobname"] = jobname
		resultMap["iolimitdevice"] = m.GetString("iolimitdevice")
		resultMap["cpushares"] = m.GetString("cpushares")
		resultMap["cpuquotaus"] = m.GetString("cpuquotaus")
		resultMap["memorylimit"] = m.GetString("memorylimit")
		resultMap["memoryswlimit"] = m.GetString("memoryswlimit")
		resultMap["ioreadlimit"] = m.GetString("ioreadlimit")
		resultMap["iowritelimit"] = m.GetString("iowritelimit")
		resultMap["command"] = m.GetString("command")
		resultMap["cronexpr"] = m.GetString("cronexpr")
		resultMap["timeout"] = m.GetString("timeout")
		resultMap["synflag"] = m.GetString("synflag")
		resultMap["whiteips"] = m.GetString("whiteips")
		resultMap["blackips"] = m.GetString("blackips")
		resultMap["killFlag"] = m.GetString("killFlag")
		results = append(results, resultMap)
		return nil
	})
	return results,err
}

func getOnceJobVersion(token string, jobName string) string {
	var version string
	query := `
			select 
				version 
			from 
				oncejobtask 
			where token='%s' and jobname='%s' and hasrun = '0' order by add_time limit 1`
	query = fmt.Sprintf(query, token, jobName)
	db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		version = m.GetString("version")
		return nil
	})
	return version
}

func GetOnceJobVersion(token string, jobName string) (string, error) {
	result := ""
	query := `
			select 
				version 
			from 
				oncejobtask 
			where token='%s' and jobname='%s' and hasrun = '0' order by add_time limit 1`
	query = fmt.Sprintf(query, token, jobName)
	err := db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		result = m.GetString("version")
		return nil
	})
	return result, err
}

func GetPackagesTask(token string) (map[string]string, error) {
	var (
		err error
		resultMap map[string]string
	)
	resultMap = make(map[string]string)
	query := `
				select 
					* 
				from 
					agent_package_task 
				where status=1 and token='%s' limit 1
	`
	query = fmt.Sprintf(query, token)
	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		resultMap["packageName"] = m.GetString("package_name")
		resultMap["packageVersion"] = m.GetString("package_version")
		return nil
	})
	if err != nil {
		return resultMap, err
	}
	return resultMap, nil
}

func WriteNodeStatus(data map[string]string) (err error) {
	var (
		hostname string
		port string
		appVersion string
		host string
		token	string
	)
	if value, ok := data["hostname"]; ok {
		hostname = value
	} else {
		return errors.New("hostname of param is null when run WriteNodeStatus")
	}
	if value, ok := data["port"]; ok {
		port = value
	} else {
		return errors.New("port of param is null when run WriteNodeStatus")
	}
	if value, ok := data["appVersion"]; ok {
		appVersion = value
	} else {
		return errors.New("appVersion of param is null when run WriteNodeStatus")
	}
	if value, ok := data["host"]; ok {
		host = value
	} else {
		return errors.New("host of param is null when run WriteNodeStatus")
	}
	if value, ok := data["token"]; ok {
		token = value
	} else {
		return errors.New("token of param is null when run WriteNodeStatus")
	}


	{
		sqlResult, err := db.ExecDb(`
                        update node_health set
                                hostname = ?,
								ip = ?,
                                http_port = ?,
                                last_seen_active = now(),
                                app_version = ?,
                                incrementing_indicator = incrementing_indicator + 1
                        where
                                token = ?
                        `,
			hostname, host, port, appVersion, token,
		)

		if err != nil {
			return log.Errore(err)
		}
		rows, err := sqlResult.RowsAffected()
		if err != nil {
			return log.Errore(err)
		}
		if rows > 0 {
			return nil
		}
	}

	// Got here? The UPDATE didn't work. Row isn't there.
	{
		sqlResult, err := db.ExecDb(`
			insert ignore into node_health
				(hostname, token, ip, first_seen_active, last_seen_active, app_version)
			values (
				?, 
				?,
				?,
				now(),
				now(), 
				?)
			`,
			hostname, token, host, appVersion,
		)
		if err != nil {
			return log.Errore(err)
		}
		rows, err := sqlResult.RowsAffected()
		if err != nil {
			return log.Errore(err)
		}
		if rows > 0 {
			return nil
		}
	}
	return nil
}

func WriteJobExecLogs(data map[string]string) (err error) {
	var (
		hostname		string
		token			string
		host			string
		jobname			string
		plantime		string
		scheduletime	string
		starttime		string
		endtime			string
		output			string
		errinfo			string
	)
	if value, ok := data["hostname"]; ok {
		hostname = value
	} else {
		return errors.New("hostname of param is null when run WriteNodeStatus")
	}
	if value, ok := data["token"]; ok {
		token = value
	} else {
		return errors.New("token of param is null when run WriteNodeStatus")
	}
	if value, ok := data["host"]; ok {
		host = value
	} else {
		return errors.New("host of param is null when run WriteNodeStatus")
	}
	if value, ok := data["jobname"]; ok {
		jobname = value
	} else {
		return errors.New("jobname of param is null when run WriteNodeStatus")
	}
	if value, ok := data["plantime"]; ok {
		plantime = value
	} else {
		return errors.New("plantime of param is null when run WriteNodeStatus")
	}
	if value, ok := data["scheduletime"]; ok {
		scheduletime = value
	} else {
		return errors.New("scheduletime of param is null when run WriteNodeStatus")
	}
	if value, ok := data["starttime"]; ok {
		starttime = value
	} else {
		return errors.New("starttime of param is null when run WriteNodeStatus")
	}
	if value, ok := data["endtime"]; ok {
		endtime = value
	} else {
		return errors.New("endtime of param is null when run WriteNodeStatus")
	}
	if value, ok := data["output"]; ok {
		result, _ :=  base64.StdEncoding.DecodeString(value)
		output = string(result)
	} else {
		return errors.New("output of param is null when run WriteNodeStatus")
	}
	if value, ok := data["errinfo"]; ok {
		result, _ :=  base64.StdEncoding.DecodeString(value)
		errinfo = string(result)
	} else {
		return errors.New("errinfo of param is null when run WriteNodeStatus")
	}

	sqlResult, err := db.ExecDb(`
                        insert into joblogs(hostname,token,ip,jobname,plantime,scheduletime,starttime,endtime,output,err) 
						values(?,?,?,?,?,?,?,?,?,?)
                        `,
			hostname, token, host, jobname, plantime, scheduletime, starttime, endtime, output, errinfo,
	)
	if err != nil {
			return log.Errore(err)
	}
	rows, err := sqlResult.RowsAffected()
	if err != nil {
			return log.Errore(err)
	}
	if rows > 0 {
			return nil
	}

	return nil
}

func SaveOnceJobStatusOrLog(data map[string]string) (err error) {
	var (
		status				string
		starttime			string
		endtime				string
		output				string
		errinfo			string

		saveStatusFlag		bool
		saveOnceJobLogFlag	bool

		token				string
		version				string
		jobname				string
	)
	if value, ok := data["saveStatusFlag"]; ok {
		if value == "1" {
			saveStatusFlag = true
		} else {
			saveStatusFlag = false
		}
	}

	if value, ok := data["saveOnceJobLogFlag"]; ok {
		if value == "1" {
			saveOnceJobLogFlag = true
		} else {
			saveOnceJobLogFlag = false
		}
	}

	if !saveStatusFlag && !saveOnceJobLogFlag {
		return errors.New("saveStatusFlag and saveOnceJobLogFlag.  One of the two parameters must be true")
	}

	if value, ok := data["status"]; ok {
		status = value
	}
	if value, ok := data["starttime"]; ok {
		starttime = value
	}
	if value, ok := data["endtime"]; ok {
		endtime = value
	}
	if value, ok := data["output"]; ok {
		output = value
	}
	if value, ok := data["errinfo"]; ok {
		errinfo = value
	}
	if value, ok := data["token"]; ok {
		token = value
	}
	if value, ok := data["version"]; ok {
		version = value
	}
	if value, ok := data["jobname"]; ok {
		jobname = value
	}

	if saveStatusFlag && !saveOnceJobLogFlag{
		update :=  "update oncejobtask set status= ? where token=? and version=? and jobname=?"
		_, err := db.ExecDb(update, status, token, version, jobname)
		 if err != nil {
			 log.Errorf("%s params (%d, %s) err %+v", update, status, token, version, jobname)
			 return err
		 }
	}

	if !saveStatusFlag && saveOnceJobLogFlag{
		_, err := db.ExecDb(`
				update 
					oncejobtask
				set hasrun='1', starttime=?, endtime=?, output=?, err=? 
				where token=? and version=? and jobname=?
			`,
			starttime, endtime, output,errinfo, token, version, jobname,
		)
		if err != nil {
			log.Errorf("Save onceJob %s result log to  oncejobtask  err: %+v", jobname, err)
			return err
		}
	}

	if saveStatusFlag && saveOnceJobLogFlag{
		_, err := db.ExecDb(`
				update 
					oncejobtask
				set status= ? , hasrun='1', starttime=?, endtime=?, output=?, err=? 
				where token=? and version=? and jobname=?
			`,
			status, starttime, endtime, output,errinfo, token, version, jobname,
		)
		if err != nil {
			log.Errorf("Save onceJob %s result log to  oncejobtask  err: %+v", jobname, err)
			return err
		}
	}

	return nil
}


func GetNonLiveIpSegment() ([]string,error) {
	var (
		query string
		err error
		results []string
	)
	results = make([]string,0)
	query = fmt.Sprintf("select * from nonliveips where filter_flag = 1")
	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		results = append(results, m.GetString("segment"))
		return nil
	})
	if err != nil {
		return results, err
	}
	return results, nil
}

func GetVariables(receiveMap map[string]string) error {
	var (
		query string
		err error
	)
	query = fmt.Sprintf("select * from variables where enable = '1'")
	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		receiveMap[m.GetString("variable")] = m.GetString("value")
		return nil
	})
	return err
}