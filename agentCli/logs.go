package agentCli

import (
        "fmt"
        "op-agent/db"
        "github.com/openark/golib/log"
        "github.com/openark/golib/sqlutils"
        "github.com/spf13/cobra"
        "strings"
)

var wideView string
var columns   string

func newLogs() *cobra.Command {
        cmd := &cobra.Command{
                Use:   "logs <JOBNAME> <HOSTIP|ALL> [LIMIT] [--o wide|short]",
                Short: "View the log information of a task",
                Long:  `Example:
                        logs test.py    //view all
                        logs test.py 192.168.1.1 10 --o wide
                        logs test.py 192.168.1.1 5 --o short
		`,
                SilenceUsage: true,
                RunE: func(cmd *cobra.Command, args []string) error {
                        var (
                        	jobName string
                        	hostIP string
                        	limit string
                        )
                        if len(args) == 0 {
                                log.Errorf("JobName must not be null")
                                return nil
                        }
                        jobName = args[0]
                        if len(args) > 1 {
                                hostIP = args[1]
                        }
                        if len(args) > 2 {
                                limit = args[2]
                        }
                        getJobLogs(jobName, hostIP, limit, wideView)
                        return nil
                },
        }
        cmd.PersistentFlags().StringVar((*string)(&wideView), "o", "", "Show details.")
        return cmd
}


func getJobLogs(jobName string, hostIP string, limit string, wide string) {

        var (
                dataLists [][]string
                hostIPS []map[string]string
        )
        dataLists = [][]string{}
        hostIPS = []map[string]string{}

        if limit == "" {
                limit = "1"
        }
        if hostIP != "" {
                hostIPS = append(hostIPS, map[string]string{"ip":hostIP, "token":""})
        } else {
                hostIPS = getAllHosts()
        }

        for _, ipMap := range hostIPS {
                query := ""
                if ipMap["token"] == "" {
                        query = fmt.Sprintf("select * from oncejobtask where jobname = '%s' and ip REGEXP '%s' order by endtime desc limit %s", jobName, ipMap["ip"], limit)
                }  else {
                        query = fmt.Sprintf("select * from oncejobtask where jobname = '%s' and token = '%s' order by endtime desc limit %s", jobName, ipMap["token"], limit)
                }

                err := db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
                        resultList := []string{}
                        if wide != "short" {
                                resultList = append(resultList, m.GetString("jobname"))
                        }
                        if wide == "wide" {
                                resultList = append(resultList, m.GetString("command"))
                                resultList = append(resultList, m.GetString("hostname"))
                        }
                        if wide != "short" {
                                resultList = append(resultList, strings.Replace(m.GetString("ip"), ",", "\n", -1))
                        }
                        if wide == "wide" {
                                resultList = append(resultList, m.GetString("starttime"))
                        }
                        if wide != "short" {
                                resultList = append(resultList, m.GetString("endtime"))
                        }
                        resultList = append(resultList, m.GetString("output"))
                        resultList = append(resultList, m.GetString("err"))
                        dataLists = append(dataLists, resultList)
                        return nil
                })
                if err != nil {
                        log.Errorf("%s err %+v", query, err)
                        return
                }
        }


        title := []string{}
        if wide == "wide" {
                title = []string{"JobName", "Command", "HostName", "HostIPS", "StartTime", "EndTime", "Output","Error"}
        } else if wide == "short" {
                title =  []string{"Output","Error"}
        } else {
                title =  []string{"JobName", "HostIPS", "EndTime", "Output","Error"}
        }
        if wide == "short" {
                for _, rowList := range dataLists {
                        fmt.Println(rowList[0], rowList[1])
                }
        } else {
                table := TableWriter(title, dataLists)
                table.Render()
        }
        return
}


func getAllHosts() (results []map[string]string) {
        var err error
        results = []map[string]string{}
        query := "select ip,token from node_health"
        if results, err = db.QueryAll(query); err != nil {
                log.Errorf("%s err %+v", query, err)
                return
        }
        return
}
