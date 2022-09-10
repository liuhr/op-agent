
package opagent

import (
	"fmt"
	"op-agent/db"
	"github.com/openark/golib/log"
	"github.com/openark/golib/sqlutils"
	"github.com/spf13/cobra"
	"strings"
)


func newJobs() *cobra.Command {
	var (
		jobName string
		wideView string
	)

	cmd := &cobra.Command{
		Use: "jobs",
		Short: "View jobs information",
		Long: `Example:
			jobs [JobName]
		`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				jobName = args[0]
			}
			if jobName != "" {
				getOneJobInfo(jobName)
			} else {
				getAllJobsInfo(wideView)
			}
			return nil
		},
	}
	cmd.PersistentFlags().StringVar((*string)(&wideView), "o", "", "Show details.")
	return cmd
}

func getOneJobInfo(jobName string) {
	var (
		err error
		query string
	)
	query = fmt.Sprintf(`select  * from jobs where jobname='%s'`, jobName)
	dataLists := [][]string{}

	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		list := []string{}
		list = append(list, "jobName:")
		list = append(list, m.GetString("jobname"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "command: ")
		list = append(list, m.GetString("command"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "Timed task expression(croneExpr): ")
		list = append(list, m.GetString("cronexpr"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "Is it a one-time task: ")
		onceFlag := ""
		if m.GetString("oncejob") == "1" {
			onceFlag = "Yes"
		} else {
			onceFlag = "No"
		}
		list = append(list, onceFlag)
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "Command execution timeout setting(timeout): ")
		list = append(list, m.GetString("timeout"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "Synchronous or asynchronous execution: ")
		syncFlag := ""
		if m.GetString("synflag") == "1" {
			syncFlag = "Synchronous"
		}  else {
			syncFlag = "Asynchronous"
		}
		list = append(list, syncFlag)
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "CGroup cpu.shares value: ")
		list = append(list, m.GetString("cpushares"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "CGroup cpu.cfs_quota_us value: ")
		list = append(list, m.GetString("cpuquotaus"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "CGroup memory.limit_in_bytes value: ")
		list = append(list, m.GetString("memorylimit"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "CGroup memory.memsw.limit_in_bytes value: ")
		list = append(list, m.GetString("memoryswlimit"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "CGroup blkio.throttle.read_bps_device value: ")
		list = append(list, m.GetString("ioreadlimit"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "CGroup blkio.throttle.write_bps_device value: ")
		list = append(list, m.GetString("iowritelimit"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "CGroup IO restricted devices: ")
		list = append(list, m.GetString("iolimitdevice"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "Whitelist of task execution(WHITELIST): ")
		list = append(list, strings.Replace(m.GetString("whiteips"), ",", "\n", -1))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "Blacklist of task execution(BLACKLIST): ")
		list = append(list, strings.Replace(m.GetString("blackips"), ",", "\n", -1))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "ADDTIME: ")
		list = append(list, m.GetString("add_time"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "LASTUPDATE: ")
		list = append(list, m.GetString("_timestamp"))
		dataLists = append(dataLists, list)

		list = []string{}
		list = append(list, "Task status(STATUS): ")
		list = append(list, m.GetString("enabled"))
		dataLists = append(dataLists, list)
		return nil
	})
	if err != nil {
		log.Errorf("%s err %+v", query, err)
		return
	}
	table := TableWriter([]string{"KeyWord", "Value",}, dataLists)
	table.Render()
	return
}

func getAllJobsInfo(wideView string)  {
	var (
		err error
		query string
	)
	query = "select  * from jobs"
	dataLists := [][]string{}
	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		resultList := []string{}
		resultList = append(resultList, m.GetString("jobname"))
		resultList = append(resultList, m.GetString("command"))
		resultList = append(resultList, m.GetString("cronexpr"))
		if m.GetString("oncejob") == "1" {
			resultList = append(resultList, "Yes")
		} else {
			resultList = append(resultList, "No")
		}
		resultList = append(resultList, m.GetString("timeout"))
		if wideView == "wide" {
			resultList = append(resultList, m.GetString("add_time"))
			resultList = append(resultList, m.GetString("_timestamp"))
		}
		if m.GetString("enabled") == "1" {
			resultList = append(resultList,"Enabled")
		} else {
			resultList = append(resultList,"Disabled")
		}
		dataLists = append(dataLists, resultList)
		return nil
	})
	if err != nil {
		log.Errorf("%s err %+v", query, err)
		return
	}
	title := []string{}
	if wideView == "wide" {
		title = []string{"jobname", "command", "cronexpr", "oncejob", "timeout","addtime", "lastupdate", "status"}
	} else {
		title = []string{"jobname", "command", "cronexpr", "oncejob", "timeout", "status"}
	}

	table := TableWriter(title, dataLists)
	table.Render()
	return
}
