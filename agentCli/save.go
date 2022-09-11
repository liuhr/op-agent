package agentCli

import (
        "errors"
        "fmt"
        "github.com/openark/golib/log"
        "github.com/openark/golib/sqlutils"
        "github.com/spf13/cobra"
        "op-agent/db"
        "strings"
        "time"
)


func newSave() *cobra.Command {
        cmd := &cobra.Command{
                Use:   "save <JOSON FILE>",
                Short: "Create a task",
                Long:  `Example:
			            save <task.json>
		`,
                SilenceUsage: true,
                RunE: func(cmd *cobra.Command, args []string) error {
                        if len(args) == 0 {
                                log.Errorf("Task parameters can not be null. Can run ./agentCli save -h for detail")
                                return nil
                        }
                        taskFile := args[0]
                        ForceRead(taskFile)
                        if TaskConfig.JobName == "" || TaskConfig.Command == "" {
                                log.Errorf("JobName and Command can not be null.")
                                return nil
                        }
                        if TaskConfig.OnceJob == 1 {
                                if err := saveOnceJobTask(); err != nil {
                                        log.Errorf("saveOnceJobTask %s err %+v", TaskConfig.JobName, err)
                                        return nil
                                }
                        }
                        if err := saveJobs(); err != nil {
                                log.Errorf("%+v", err)
                                return nil
                        }
                        return nil
                },
        }
        return cmd
}


func saveJobs() error {
        var (
        	whiteips string
        	blackips string
        )
        if TaskConfig.OnceJob == 0 {
                whiteips = strings.Join(TaskConfig.WhiteHosts,",")
                blackips = strings.Join(TaskConfig.BlackHosts, ",")
        }
        replace := `replace 
                        into jobs(jobname, command, cronexpr, 
                                oncejob, timeout, synflag,
                                whiteips, blackips, cpushares, 
                                cpuquotaus, memorylimit, memoryswlimit,
                                ioreadlimit, iowritelimit, iolimitdevice, enabled) 
                        values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        `
        sqlResult, err := db.ExecDb(replace,
                TaskConfig.JobName, TaskConfig.Command, TaskConfig.CronExpr, TaskConfig.OnceJob,
                TaskConfig.Timeout, TaskConfig.SynFlag, whiteips, blackips, TaskConfig.CPUShares, TaskConfig.CPUQuotaUs,
                TaskConfig.MemoryLimit, TaskConfig.MemorySwLimit, TaskConfig.IOReadLimit, TaskConfig.IOWriteLimit, TaskConfig.IOLimitDevice, TaskConfig.Enabled)

        if err != nil {
                return errors.New(fmt.Sprintf("saveJobs err %s %+v %+v", replace, TaskConfig, err))
        }
        rows, err := sqlResult.RowsAffected()
        if err != nil {
                return  errors.New(fmt.Sprintf("saveJobs %s config err %+v", TaskConfig.JobName, err))
        }
        if rows < 1 {
                return errors.New(fmt.Sprintf("saveJobs %s config to meta did not affect.", TaskConfig.JobName))
        }
        return nil
}

func saveOnceJobTask() error {
        var (
                noActiveHost  []string
                onceJobTaskHostList []map[string]string
        )
        noActiveHost = []string{}
        onceJobTaskHostList = []map[string]string{}
        query := "select count(1) as count from jobs where jobname = ?"
        err := db.QueryDB(query, sqlutils.Args(TaskConfig.JobName),func(m sqlutils.RowMap) error {
                if m.GetInt("count") == 0 {
                        db.ExecDb("delete from oncejobtask where jobname = ?", TaskConfig.JobName)
                }
                return nil
        })
        if err != nil {
                return errors.New(fmt.Sprintf("Query %s jobs history record err %+v", TaskConfig.JobName ,err))
        }

        allActiveHosts, err := GetAllActiveHosts()
        if err != nil {
                return errors.New(fmt.Sprintf("Get all active agents err %+v", err))
        }
        if len(allActiveHosts) == 0 {
                return errors.New(fmt.Sprintf("Found no active agents, Pls check."))
        }
        if  len(TaskConfig.WhiteHosts) == 0 && len(TaskConfig.BlackHosts) == 0 {
                onceJobTaskHostList = allActiveHosts
        }
        if len(TaskConfig.WhiteHosts) == 0 && len(TaskConfig.BlackHosts) != 0  {
                for _, ip := range TaskConfig.BlackHosts {
                        for _, activeAgent := range allActiveHosts {
                                if len(activeAgent) == 0 {
                                        continue
                                }
                                if strings.Contains(activeAgent["ip"], ip) {
                                        activeAgent["BlackFlag"] = ""
                                }
                        }
                }
                for _, activeAgent := range allActiveHosts {
                        if _, ok := activeAgent["BlackFlag"]; !ok {
                                onceJobTaskHostList = append(onceJobTaskHostList, activeAgent)
                        }
                }
        }
        if len(TaskConfig.WhiteHosts) != 0 && len(TaskConfig.BlackHosts) == 0 {
                for _, ip := range TaskConfig.WhiteHosts {
                        findFlag := false
                        for _, activeAgent := range allActiveHosts {
                                if len(activeAgent) == 0 {
                                        continue
                                }
                                if strings.Contains(activeAgent["ip"], ip) {
                                        onceJobTaskHostList = append(onceJobTaskHostList, activeAgent)
                                        findFlag = true
                                }
                        }
                        if !findFlag {
                                noActiveHost = append(noActiveHost, ip)
                        }
                }
        }
        if len(TaskConfig.WhiteHosts) != 0 && len(TaskConfig.BlackHosts) != 0 {
                newWhiteHosts := []string{}
                whiteHosts := strings.Join(TaskConfig.BlackHosts,",")
                for _, ip := range TaskConfig.WhiteHosts {
                        if !strings.Contains(whiteHosts, ip) {
                               newWhiteHosts = append(newWhiteHosts, ip)
                        }
                }
                for _, ip := range newWhiteHosts {
                        findFlag := false
                        for _, activeAgent := range allActiveHosts {
                                if len(activeAgent) == 0 {
                                        continue
                                }
                                if strings.Contains(activeAgent["ip"], ip) {
                                        onceJobTaskHostList = append(onceJobTaskHostList, activeAgent)
                                        findFlag = true
                                }
                        }
                        if !findFlag {
                                noActiveHost = append(noActiveHost, ip)
                        }
                }
        }
        if len(noActiveHost) != 0 {
                log.Errorf("The agent in these hosts are not active, Pls check.")
                for _, ip := range noActiveHost {
                        log.Errorf(ip)
                }
                return nil
        }
        timeUnix := time.Now().Unix()
        for _, onceJobHost := range onceJobTaskHostList {
                log.Infof("saveOnceJobTask of %s for %s", TaskConfig.JobName, onceJobHost["ip"])
                insert := "insert ignore into oncejobtask (hostname, token, ip, jobname, command, version) values(?, ?, ?, ?, ?, ?)"
                _, err := db.ExecDb(insert,
                                        onceJobHost["hostname"], onceJobHost["token"], onceJobHost["ip"],
                                        TaskConfig.JobName, TaskConfig.Command, timeUnix,
                                )
                if err != nil {
                        return errors.New(fmt.Sprintf("saveOnceJobTask of %s %+v err %+v", insert, TaskConfig, err))
                }
                /*rows, err = sqlResult.RowsAffected()
                if err != nil {
                        return  errors.New(fmt.Sprintf("saveOnceJobTask of %s config err %+v", TaskConfig.JobName, err))
                }
                if rows < 1 {
                        return errors.New(fmt.Sprintf("saveOnceJobTask %s config to meta did not affect.", TaskConfig.JobName))
                }*/
        }
        log.Warningf("Pls take the version of onceJob %s you can use it to check its status VERSION: %d", TaskConfig.JobName, timeUnix)
        return nil
}
