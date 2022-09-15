package agentCli

import (
	"fmt"
	"strings"
	"sync"

	"github.com/asmcos/requests"
	"github.com/openark/golib/log"
	"github.com/openark/golib/sqlutils"

	"op-agent/config"
	"op-agent/db"
	"op-agent/util"
)

func TakeAgentsStatus(host string, wide string) [][]string {
	var wg, wgHandleData sync.WaitGroup
	dataLists := make([][]string, 0)
	agentsOriginalInfoChan := make(chan []string, 100)
	agentsHandledInfoChan := make(chan []string, 100)
	wg.Add(1)
	go getHandledInfoFromChan(dataLists, agentsHandledInfoChan, &wg)

	for i := uint(1); i <= 100; i++ {
		wgHandleData.Add(1)
		go handleOriginalInfo(agentsOriginalInfoChan, agentsHandledInfoChan, &wgHandleData)
	}

	if err := takeAgentsInfoFromBackend(host, wide, agentsHandledInfoChan); err != nil {
		log.Error("takeAgentsInfoFromBackend err: %+v", err)
	}
	wgHandleData.Wait()
	close(agentsHandledInfoChan)
	wg.Wait()
	return dataLists
}

func takeAgentsInfoFromBackend(host string,  wide string, agentsOriginalInfoChan chan []string) error {
	var (
		err error
		query string
	)
	defer close(agentsOriginalInfoChan)

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
	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		resultList := make([]string, 0)
		resultList = append(resultList, m.GetString("hostname"))
		resultList = append(resultList, m.GetString("ip"))
		resultList = append(resultList, m.GetString("last_seen_from_now_minutes"))
		resultList = append(resultList, m.GetString("app_version"))
		resultList = append(resultList, m.GetString("first_seen_active"))
		resultList = append(resultList, m.GetString("last_seen_active"))
		if host != "" || wide == "wide" {
			hostPackages := make([]string, 0)
			allNewPackages := make([]string, 0)
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
		agentsOriginalInfoChan <- resultList
		return nil
	})

	return err
}

func handleOriginalInfo(agentsOriginalInfoChan chan []string, agentsHandledInfoChan chan []string, wg *sync.WaitGroup) {
	defer wg.Done()
	for data := range agentsOriginalInfoChan {
		if len(data) == 0 {
			continue
		}
		if util.ConvStrToInt(data[2]) > 10 { //data[2] stored 'last_seen_from_now_minutes'
			if GetNodeStatusFromApi(data[1]) { //data[1] stored 'ip'
				data[2] = "Ready"
			} else {
				data[2] = "NotReady"
			}
		}
		agentsHandledInfoChan <- data
	}
}

func getHandledInfoFromChan(dataLists [][]string, agentsHandledInfoChan chan []string, wg *sync.WaitGroup) {
	defer wg.Done()
	for data := range agentsHandledInfoChan {
		dataLists = append(dataLists, data)
	}
}

func GetNodeStatusFromApi(ips string) bool {
	for _, ip := range strings.Split(ips, ",") {
		req := requests.Requests()
		req.SetTimeout(2)
		versionApi := fmt.Sprintf("http://%s:%d/api/version", ip, config.Config.OpAgentPort)
		resp, err := req.Get(versionApi, requests.Auth{config.Config.OpAgentUser, config.Config.OpAgentPass})
		if err != nil {
			continue
		}
		resultStr := resp.Text()
		if strings.Contains(resultStr, "OK") {
			return true
		}
	}
	return false
}


