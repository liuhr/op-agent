package common

import (
	"github.com/openark/golib/log"

	"op-agent/db"
)

type Package struct {
	PackageName string
	PackageVersion string
	DeployDir string
	Md5sum	string
	PackageDesc string
}

type AgentNodeSpec struct {
	HostName string
	Token	string
	HostIps string
	HttpPort int
	AppVersion string
}

type AgentNodePackage struct {
	AgentNodeSpec *AgentNodeSpec
	Packages map[string]*Package
}

func SaveAgentNodeInfoToBackend(item map[string]string) error {
	if item["newToken"] == "" {
		item["newToken"] = item["token"]
	}

	if item["err"] != "" {
		_, err := db.ExecDb(`
			update node_health set
				err = ?
			where
				token = ?
			`,
			item["err"], item["token"],
		)
		return err
	}

	{
		sqlResult, err := db.ExecDb(`
			update node_health set
				hostname = ?,
				ip = ?,
				token = ?,
				last_seen_active = now(),
				app_version = ?,
				err = ?,
				incrementing_indicator = incrementing_indicator + 1
			where
				token = ?
			`,
			item["hostname"], item["ip"], item["newToken"], item["app_version"], item["err"]+" "+item["getPackagesInfo"]+" "+item["getJobsInfo"], item["token"],
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
				(hostname, token, ip, http_port, last_seen_active, app_version, first_seen_active, err)
			values ( ?, ?, ?, ?, now(), ?, now(), ?)
			`,
			item["hostname"], item["newToken"], item["ip"], item["port"], item["app_version"], item["err"]+" "+item["getPackagesInfo"]+" "+item["getJobsInfo"],
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
