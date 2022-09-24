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

func TakeAgentsStatus(host string, wide string) ([]map[string]string, error) {
	var (
		err error
		wg, wgHandleData sync.WaitGroup
	)
	agentsOriginalInfoChan := make(chan map[string]string, 100)
	agentsHandledInfoChan := make(chan map[string]string, 100)
	resultsChan := make(chan []map[string]string, 1)
	wg.Add(1)
	go getHandledInfoFromChan(resultsChan, agentsHandledInfoChan, &wg)

	for i := uint(1); i <= 100; i++ {
		wgHandleData.Add(1)
		go handleOriginalInfo(agentsOriginalInfoChan, agentsHandledInfoChan, &wgHandleData)
	}

	if err = takeAgentsInfoFromBackend(host, wide, agentsOriginalInfoChan); err != nil {
		log.Error("takeAgentsInfoFromBackend err: %+v", err)
	}
	wgHandleData.Wait()
	close(agentsHandledInfoChan)
	wg.Wait()
	dataLists := <-resultsChan
	return dataLists, err
}

func takeAgentsInfoFromBackend(host string,  wide string, agentsOriginalInfoChan chan map[string]string) error {
	var (
		err error
		query string
	)
	defer close(agentsOriginalInfoChan)

	if host != "" {
		query = `select  
					hostname,token,ip,http_port,app_version,last_seen_active,first_seen_active, 
        			TIMESTAMPDIFF(MINUTE,last_seen_active,now()) as last_seen_from_now_minutes,active_flag 
				from 
					node_health where hostname='%s' or ip REGEXP '%s'`
		query = fmt.Sprintf(query, host,host)
	} else {
		query = `select 
					hostname,token,ip,http_port,app_version,last_seen_active,first_seen_active, 
        			TIMESTAMPDIFF(MINUTE,last_seen_active,now()) as last_seen_from_now_minutes,active_flag 
				from node_health`
	}
	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		resultMap := make(map[string]string, 0)
		resultMap["hostname"] = m.GetString("hostname")
		resultMap["ip"] = m.GetString("ip")
		resultMap["token"] = m.GetString("token")
		resultMap["port"] = m.GetString("http_port")
		resultMap["active_flag"] = m.GetString("active_flag")
		resultMap["last_seen_from_now_minutes"] = m.GetString("last_seen_from_now_minutes")
		resultMap["app_version"] = m.GetString("app_version")
		resultMap["first_seen_active"] = m.GetString("first_seen_active")
		resultMap["last_seen_active"] = m.GetString("last_seen_active")
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
			resultMap["hostPackages"] = strings.Join(hostPackages, "\n")
			resultMap["allNewPackages"] = strings.Join(allNewPackages, "\n")
		}
		agentsOriginalInfoChan <- resultMap
		return nil
	})

	return err
}

func handleOriginalInfo(agentsOriginalInfoChan chan map[string]string, agentsHandledInfoChan chan map[string]string, wg *sync.WaitGroup) {
	defer wg.Done()
	for data := range agentsOriginalInfoChan {
		if len(data) == 0 {
			continue
		}
		if util.ConvStrToInt(data["last_seen_from_now_minutes"]) > 10 {
			if GetNodeStatusFromApi(data["ip"]) {
				data["status"] = "Ready"
			} else {
				data["status"] = "NotReady"
			}
		} else {
			data["status"] = "Ready"
		}
		agentsHandledInfoChan <- data
	}
}

func getHandledInfoFromChan(resultsChan chan []map[string]string, agentsHandledInfoChan chan map[string]string, wg *sync.WaitGroup) {
	dataLists := make([]map[string]string, 0)
	defer wg.Done()
	for data := range agentsHandledInfoChan {
		dataLists = append(dataLists, data)
	}
	resultsChan <- dataLists
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


