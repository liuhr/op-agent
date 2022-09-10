package managePlugin

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/asmcos/requests"
	oraft "op-agent/raft"
	"github.com/openark/golib/log"
	"math/rand"
	"strings"
	"time"

	"op-agent/common"
	"op-agent/config"
	"op-agent/db"
	"op-agent/util"
	"github.com/openark/golib/sqlutils"
)


type PackageController struct {}

var (
	nodeAgentQueue      *Queue
	AgentPackageControl *PackageController
)

func (controller *PackageController) ContinueGetNodesAgentsSpec() {
	go func() {
		continuesTick := time.Tick(time.Duration(rand.Intn(2) +
			int(config.Config.AgentNodesPullPackagesIntervalSeconds)) * time.Second)
		for range continuesTick {
			if oraft.IsRaftEnabled() {
				if !oraft.IsLeader() {
					continue
				}
			}
			nodeQueue := CreateOrReturnQueue("DEFAULT")
			if nodeQueue.QueueLen() != 0 {
				log.Warningf("It is detected that there are nodes in the nodeQueue. Ignore this loop.")
				continue
			}
			controller.GetAllAgentsDesc()
		}
	}()
}

func (controller *PackageController) ContinueUpdatePackagesThroughAgentApi() {
	go func() {
		continuesTick := time.Tick(time.Duration(2) * time.Second)
		for range continuesTick {
			if oraft.IsRaftEnabled() {
				if !oraft.IsLeader() {
					continue
				}
			}
			controller.UpdatePackageThroughAgentApi()
		}
	}()
}

func (controller *PackageController) UpdatePackageThroughAgentApi()  error {
	query := fmt.Sprintf("select a.*, h.http_port  from agent_package_task a left join node_health h on a.token = h.token where a.status=1 and a.package_schedule_time not between date_add(now(), interval - 5 second) and now()  group by a.token")
	err := db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		var requestErr error
		id := m.GetString("id")
		hostIps := m.GetString("agent_ips")
		hostPort := m.GetString("http_port")
		if hostPort == "0" {
			return nil
		}
		requestFlag := false
		go func() {
			for _, ip := range strings.Split(hostIps, ",") {
				if !util.IsIP(ip) {
					continue
				}
				controller.UpdatePackageScheduleTime(id)
				if err := controller.QuestAgentPackageApi(ip, hostPort); err == nil {
					requestFlag = true
					break
				} else {
					requestErr = err
				}
			}
			if !requestFlag {
				log.Errorf("Agent hostAddr %s:%s UpdatePackageThroughAgentApi err %+v", hostIps, hostPort, requestErr)
				controller.UpdateAgentPackageTaskErrMsg(id, "Run UpdatePackageThroughAgentApi err")
			}
		}()
		return nil
	})
	return err
}

func  (controller *PackageController) UpdatePackageScheduleTime(id string) {
	db.ExecDb("update agent_package_task set package_schedule_time = now() where id = ?", id )
}

func (controller *PackageController) UpdateAgentPackageTaskErrMsg(id string, errInfo string) {
	db.ExecDb("update agent_package_task set fail_reason = ? where id = ?", errInfo, id )
}

func (controller *PackageController) QuestAgentPackageApi(host string, port string) error {
	req := requests.Requests()
	defer req.Close()
	agentApi := fmt.Sprintf("http://%s:%s/api/update-agent-package",host, port)
	log.Infof("Request agentApi %s", agentApi)
	resp, err := req.Get(agentApi, requests.Auth{config.Config.OpAgentUser, config.Config.OpAgentPass})
	if err != nil {
		return err
	}
	resultStr := resp.Text()
	if strings.Contains(resultStr, "OK") {
		return nil
	} else {
		errors.New(resultStr)
	}
	return nil
}

func (controller *PackageController) ContinueGetNodesAgentsPackageDesc() {
	nodeAgentQueue = CreateOrReturnQueue("DEFAULT")
	for i:= uint(0); i < config.Config.DealWithAgentsDescMaxConcurrency; i++ {
		go func() {
			for {
				if oraft.IsRaftEnabled() {
					if !oraft.IsLeader() {
						time.Sleep(time.Second * 1)
						continue
					}
				}
				newAgentNodePackage := &common.AgentNodePackage{}
				nodeAgentSpec := nodeAgentQueue.Consume()
				// nodeAgentSpec == {db-host-name01 8e6fcfba5cb2ff0c24bba492268 172.16.123.1 8080 1.0.0}
				agentNodePackagesMap := make(map[string]*common.Package)
				query := fmt.Sprintf("select * from agent_package_info where token='%s'",nodeAgentSpec.Token)
				db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
					agentNodePackagesMap[m.GetString("package_name")] = &common.Package{
						PackageName:    m.GetString("package_name"),
						PackageVersion: m.GetString("package_version"),
					}
					return nil
				})

				packagesMap := make(map[string]*common.Package)
				query = fmt.Sprintf("select package_name,max(package_version) as package_version,md5sum,deploydir from package_info group by package_name")
				db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
					packagesMap[m.GetString("package_name")] = &common.Package{
						PackageName:    m.GetString("package_name"),
						PackageVersion: m.GetString("package_version"),
						Md5sum: m.GetString("md5sum"),
						DeployDir: m.GetString("deploydir"),
					}
					return nil
				})

				newPackageMap := make(map[string]*common.Package)
				for packageName, packageSpec := range packagesMap {
					if packageSpecInNode, ok := agentNodePackagesMap[packageName]; !ok {
						newPackageMap[packageName] = packageSpec
					} else {
						if packageSpec.PackageVersion != packageSpecInNode.PackageVersion {
							newPackageMap[packageName] = packageSpec
						}
					}
				}
				if len(newPackageMap) != 0 {
					newAgentNodePackage.AgentNodeSpec = &nodeAgentSpec
					newAgentNodePackage.Packages = newPackageMap
					if err := controller.saveTaskToMeta(newAgentNodePackage); err != nil {
						log.Errorf("packageTask.saveTaskToMeta err: %+v", err)
					}
				}

				nodeAgentQueue.Release(nodeAgentSpec)
			}
		}()
	}
}


func (controller *PackageController) saveTaskToMeta(nodePackage *common.AgentNodePackage) error {
	var (
		sqlResult sql.Result
		err error
	)
	for packageName, packageDesc := range nodePackage.Packages {
		status := "0"
		hostName := nodePackage.AgentNodeSpec.HostName
		token := nodePackage.AgentNodeSpec.Token
		agentIps := nodePackage.AgentNodeSpec.HostIps
		packageVersion := packageDesc.PackageVersion
		deployDir := packageDesc.DeployDir
		insert := `
				insert ignore into agent_package_task
				(hostname, token, agent_ips, package_name, package_version, deploydir, status, ctime)
			values
				(?, ?, ?, ?, ?, ?, ?, now())
			`
		sqlResult, err = db.ExecDb(insert,
			hostName, token, agentIps, packageName, packageVersion, deployDir, status,
		)
		if err != nil {
			return fmt.Errorf("%s %+v", insert, err)
		}
		if _, err = sqlResult.RowsAffected(); err != nil {
			return fmt.Errorf("%s %+v",insert ,err)
		}
	}
	return nil
}

func (controller *PackageController) GetAllAgentsDesc() {
	var (
		query string
	)
	nodeQueue := CreateOrReturnQueue("DEFAULT")
	query = fmt.Sprintf("select * from node_health where last_seen_active  between date_add(now(), interval - %d second) and now()", config.Config.LastSeenTimeoutSeconds)
	db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		agentNodeSpec := common.AgentNodeSpec{
			HostName: m.GetString("hostname"),
			Token: m.GetString("token"),
			HostIps: m.GetString("ip"),
			HttpPort: util.ConvStrToInt(m.GetString("http_port")),
			AppVersion: m.GetString("app_version"),
		}
		nodeQueue.Push(agentNodeSpec)
		return nil
	})
}

func RunAgentPackageControl() {
	AgentPackageControl.ContinueGetNodesAgentsSpec()
	AgentPackageControl.ContinueGetNodesAgentsPackageDesc()
	//AgentPackageControl.ContinueUpdatePackagesThroughAgentApi()
}
