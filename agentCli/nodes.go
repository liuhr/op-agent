package agentCli

import (
	"fmt"
	"github.com/asmcos/requests"
	"op-agent/config"
	"strings"

	"github.com/openark/golib/log"
	"github.com/openark/golib/sqlutils"
	"github.com/spf13/cobra"

	"op-agent/db"
)

var wide string

func newNodes() *cobra.Command {
	var (
		host string
	)
	cmd := &cobra.Command{
		Use: "nodes [IP|HOSTNAME] [--o wide]",
		Short: "View nodes information",
		Long: `Example:
			nodes [IP|HOSTNAME] [--o wide]
		`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				host = args[0]
			}
			if err := GetNodesStatus(host); err != nil {
				log.Errorf("GetNodesStatus('%s') err: %s+v", host, err)
			}
			return nil
		},
	}
	cmd.PersistentFlags().StringVar((*string)(&wide), "o", "", "Show details.")
	return cmd
}


func GetNodesStatus(host string) error {
	var (
		err error
		query string
	)
	if host != "" {
		query = `select  
					hostname,token,ip,app_version,last_seen_active,first_seen_active,
					TIMESTAMPDIFF(MINUTE,last_seen_active,now()) as last_seen_from_now_minutes
				from 
					node_health where hostname='%s' or ip REGEXP '%s' order by last_seen_active desc`
		query = fmt.Sprintf(query, host,host)
	} else {
		query = `select 
					hostname,token,ip,app_version,last_seen_active,first_seen_active, 
					TIMESTAMPDIFF(MINUTE,last_seen_active,now()) as last_seen_from_now_minutes 
				from node_health order by last_seen_active desc`
	}
	dataLists := [][]string{}
	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		resultList := []string{}
		resultList = append(resultList, m.GetString("hostname"))
		resultList = append(resultList, strings.Replace(m.GetString("ip"),",","\n",-1))
		if m.GetInt("last_seen_from_now_minutes") > 10 {
			if GetNodeStatusFromApi(m.GetString("ip")) {
				resultList = append(resultList, "Ready")
			} else {
				resultList = append(resultList, "NotReady")
			}
		} else {
			resultList = append(resultList, "Ready")
		}
		resultList = append(resultList, m.GetString("app_version"))
		resultList = append(resultList, m.GetString("first_seen_active"))
		resultList = append(resultList, m.GetString("last_seen_active"))
		if host != "" || wide == "wide" {
			hostPackages := []string{}
			allNewPackages := []string{}
			query := fmt.Sprintf(`select 
											concat(package_name,' ', package_version) as packageinfo 
										 from agent_package_info where token='%s' order by package_name`,
										 m.GetString("token"))
			rowsMap, _ := db.QueryAll(query)
			for _, row := range rowsMap {
				hostPackages = append(hostPackages, row["packageinfo"])
			}
			query = fmt.Sprintf(`select 
											concat(package_name, ' ', max(package_version)) as packageinfo  
										from  package_info group by package_name order by package_name`)
			rowsMap, _ = db.QueryAll(query)
			for _, row := range rowsMap {
				allNewPackages = append(allNewPackages, row["packageinfo"])
			}
			resultList = append(resultList, strings.Join(hostPackages, "\n"))
			resultList = append(resultList, strings.Join(allNewPackages, "\n"))
		}
		dataLists = append(dataLists, resultList)
		return nil
	})
	if err != nil {
		return err
	}

	title := []string{}
	if host != "" || wide == "wide" {
		title = []string{"HostName", "IP", "Status", "Version", "FirstSeen", "LastSeen", "Packages-On-This-Host", "All-Packages"}
	} else {
		title = []string{"HostName", "IP", "Status", "Version", "FirstSeen", "LastSeen"}
	}
	table := TableWriter(title, dataLists)
	table.Render()
	return nil
}

func GetNodeStatusFromApi(ips string) bool {
	for _, ip := range strings.Split(ips, ",") {
		req := requests.Requests()
		req.SetTimeout(2)
		versionApi := fmt.Sprintf("http://%s:%d/api/version", ip, config.Config.OpAgentPort)
		log.Infof(versionApi)
		resp, _ := req.Get(versionApi, requests.Auth{config.Config.OpAgentUser, config.Config.OpAgentPass})
		resultStr := resp.Text()
		if strings.Contains(resultStr, "OK") {
			return true
		}
	}
	return false
}