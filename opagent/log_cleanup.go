package opagent

import (
	"database/sql"
	"op-agent/config"
	"op-agent/db"
	"github.com/openark/golib/log"
)

func cleanupJobLogsOfEachHostOrAllHost(host string) {
	var (
		sqlResult sql.Result
		err error
	)
	for {
		if host != "" {
			sqlResult, err = db.ExecDb(`
							delete 
								from joblogs 
							where 
								ip=? and _timestamp <  now() - interval ? day limit 10
							`, host, config.Config.NumberOfJobLogDaysKeepPerHost,
			)
		} else {
			sqlResult, err = db.ExecDb(`
							delete 
								from joblogs 
							where 
								_timestamp <  now() - interval ? day limit 10
							`, config.Config.NumberOfJobLogDaysKeepInactiveHost,
			)
		}
		if err != nil {
			log.Errorf("cleanupJobLogsOfEachHost err %+v", err)
			return
		}
		rows, err := sqlResult.RowsAffected()
		if err != nil {
			log.Errorf("cleanupJobLogsOfEachHost RowsAffected err %+v", err)
			return
		}
		if rows < 1 {
			return
		}
	}
	return
}


func CleanupJobLogsOfAllHosts() {
	var (
		err error
		activeHosts []map[string]string
	)
	if activeHosts, err = GetAllActiveHosts(); err != nil {
		log.Errorf("GetAllActiveHosts err %+v", err)
		return
	}
	for _, hostMap := range activeHosts {
		cleanupJobLogsOfEachHostOrAllHost(hostMap["ip"])
	}

	cleanupJobLogsOfEachHostOrAllHost("")
}
