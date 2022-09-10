package plugin

import (
	"database/sql"
	"op-agent/config"
	"op-agent/db"
	oraft "op-agent/raft"
	"github.com/openark/golib/log"
	"github.com/openark/golib/sqlutils"
	"time"
)

type WatchPackageTask struct {}

var watchPackageTask *WatchPackageTask



func (watchPackageTask *WatchPackageTask) setTaskConcurrency() {
	var (
		runningCount int
		sqlResult sql.Result
		err error
	)
	query := "select count(1) count from agent_package_task where status in ('1','2')"
	db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		runningCount =  m.GetInt("count")
		return nil
	})

	update := `
			update 
					agent_package_task set status=1  
			where 
					status=0 and token 
					not in (select token from  (select token from agent_package_task  where status in ('1','2')) a) limit ?
	`
	_, err = db.ExecDb(update, int(config.Config.AgentDownloadPackageMaxConcurrency) - runningCount)
	if err != nil {
		log.Errorf( "%s %+v",update,err)
	}

	replaceInto := `
		REPLACE INTO 
			agent_package_info (hostname,token,agent_ips,package_name,package_version,deploydir,status) 
		SELECT 
			hostname, token, agent_ips, package_name, package_version, deploydir, status 
		FROM
			agent_package_task where status = '3'
	`
	if sqlResult, err = db.ExecDb(replaceInto); err != nil {
		log.Errorf( "%s %+v", replaceInto, err)
		return
	}
	if _, err = sqlResult.RowsAffected(); err != nil {
		log.Errorf("%s %+v",replaceInto ,err)
		return
	}

	deleteSql := "delete from agent_package_task where status = '3'"
	if _, err = db.ExecDb(deleteSql); err != nil {
		log.Errorf("%s %+v", deleteSql, err)
	}

}

func (watchPackageTask *WatchPackageTask) continuesWatchPackageTask() {
	go func() {
		continuesTick := time.Tick(time.Duration(config.WatchPackageTaskStatusSeconds) * time.Second)
		for range continuesTick {
			if oraft.IsRaftEnabled() {
				if !oraft.IsLeader() {
					continue
				}
			}
			watchPackageTask.setTaskConcurrency()
		}
	}()
}

func RunWatchPackageTask() {
	watchPackageTask.continuesWatchPackageTask()
}
