package manager

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/asmcos/requests"
	"github.com/outbrain/golib/log"
	"op-agent/agentCli"
	"op-agent/common"
	"op-agent/config"
	"op-agent/process"
	oraft "op-agent/raft"
	"op-agent/util"
	"strings"
	"time"
)

type AgentWatcher struct {}

var (
	agentWatch			*AgentWatcher
)

type AgentNodeInfoRespond struct {
	Code		string
	Message		string
	Details		map[string]string
}


func (agentWatch *AgentWatcher) GenerateAgentWatcherLoop() {
	var (
		timeTick time.Duration
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
	)

	scheduleTickNum, err :=  util.TakeRandFromList(config.Config.DiscoverOpAgentIntervalLists)
	if err != nil {
		scheduleTickNum = 30
	}
	timeTick = time.Duration(int64(scheduleTickNum)) * time.Second
	scheduleTimer = time.NewTimer(timeTick)

	for {
		select {
		case <- scheduleTimer.C:
		}
		scheduleAfter = agentWatch.GenerateAgentQueue()
		scheduleTimer.Reset(scheduleAfter)
	}
}


func (agentWatch *AgentWatcher) GenerateAgentQueue() time.Duration {
	var (
		scheduleTickNum int
		err error
	)
	schedulerTickFunc := func() time.Duration {
		if scheduleTickNum, err =  util.TakeRandFromList(config.Config.DiscoverOpAgentIntervalLists); err != nil {
			return time.Duration(30) * time.Second
		}
		return time.Duration(int64(scheduleTickNum)) * time.Second
	}

	if oraft.IsRaftEnabled() {
		if !oraft.IsLeader() {
			return schedulerTickFunc()
		}
	}
	agentWatch.pushAgentNodesToQueue()
	return schedulerTickFunc()
}


func (agentWatch *AgentWatcher) pushAgentNodesToQueue() {
	queueName := "discoverAgent"
	agentNodeQueue := agentCli.CreateOrReturnQueue(queueName)
	if agentNodeQueue.QueueLen() != 0 {
		log.Warningf("It is detected that there are agent nodes in the agentNodeQueue. Ignore this loop.")
		return
	}
	agentHosts, err := agentCli.GetOutDatedAgentHosts()
	if err != nil {
		log.Errorf("Run agentCli.GetOutDatedAgentHosts err: %+v", err)
		return
	}
	for _, agentNode := range agentHosts {
		node := agentCli.AgentNode{Hostname: agentNode["hostname"], IP: agentNode["ip"], Token: agentNode["token"], Port: util.ConvStrToInt(agentNode["port"])}
		agentNodeQueue.Push(node)
	}
}

func (agentWatch *AgentWatcher) ConcurrencyWatchAgentWatcherQueue() {
	queueName := "discoverAgent"
	agentNodeQueue := agentCli.CreateOrReturnQueue(queueName)
	for i:= uint(0); i < config.Config.DiscoverOpAgentConcurrency; i++{
		go func() {
			for {
				var (
					httpPort int
					err error
					resultStr	string
					packageTask map[string]string
					allJobs []map[string]string
					resp *requests.Response
				)
				postData := requests.Datas{}

				nodeAgent := agentNodeQueue.Consume()

				//Get packages change task
				packageTask, err = agentCli.GetPackagesTask(nodeAgent.Token)
				if err != nil {
					postData["GetPackagesTaskStatus"] = base64.StdEncoding.EncodeToString([]byte(err.Error()))
				} else {
					postData["GetPackagesTaskStatus"] = ""
				}
				result,  _ := json.Marshal(packageTask)
				postData["PackageTask"] = string(result)

				//Get all jobs
				allJobs, err = agentCli.GetAllJobs(nodeAgent.Token)
				if err != nil {
					postData["GetAllJobsStatus"] = base64.StdEncoding.EncodeToString([]byte(err.Error()))
				} else {
					postData["GetAllJobsStatus"] = ""
				}
				result, _ = json.Marshal(allJobs)
				postData["allJobs"] = string(result)

				//Get variables
				if err = agentCli.GetVariables(postData); err != nil {
					postData["getVariablesStatus"] = base64.StdEncoding.EncodeToString([]byte(err.Error()))
				}

				//Get raft available nodes
				availableNodeList := make([]string,0)
				for _, node := range process.AvailableNodes {
					availableNodeList = append(availableNodeList, node.HostIp)
				}
				postData["raftAvailableNodes"] = strings.Join(availableNodeList, ",")

				newItem := map[string]string{
					"hostname": nodeAgent.Hostname,
					"ip": nodeAgent.IP,
					"token": nodeAgent.Token,
					"port": string(nodeAgent.Port),
					"newToken": "",
					"app_version": "",
					"last_seen_active": "",
					"getPackagesInfo": "",
					"getJobsInfo": "",
					"err": "",
				}
				opAgentDataReceiveApiEndPoint := config.Config.OpAgentDataReceiveApiEndPoint
				agentUser := config.Config.OpAgentUser
				agentPass := config.Config.OpAgentPass
				ipList := strings.Split(nodeAgent.IP, ",")
				if len(ipList) == 0 {
					continue
				}
				if nodeAgent.Port != 0 {
					httpPort = config.Config.OpAgentPort
				} else {
					httpPort = nodeAgent.Port
				}
				for _, ip := range ipList {
					var receivedData AgentNodeInfoRespond
					req := requests.Requests()
					req.SetTimeout(5)
					apiAddr := fmt.Sprintf("http://%s:%d%s", ip, httpPort, opAgentDataReceiveApiEndPoint)
					resp, err = req.PostJson(apiAddr, requests.Auth{agentUser, agentPass}, postData)
					if err != nil {
						continue
					}
					resultStr = resp.Text()
					if !strings.Contains(resultStr, "OK") {
						continue
					} else {
						if err = json.Unmarshal([]byte(resultStr), &receivedData); err != nil {
							continue
						}
						newItem["hostname"] = receivedData.Details["hostname"]
						newItem["ip"] = receivedData.Details["ip"]
						newItem["newToken"] = receivedData.Details["token"]
						newItem["app_version"] = receivedData.Details["app_version"]
						newItem["getPackagesInfo"] = receivedData.Details["getPackagesInfo"]
						newItem["getJobsInfo"] = receivedData.Details["getJobsInfo"]
						break
					}
				}
				if !strings.Contains(resultStr, "OK") {
					newItem["err"] = fmt.Sprintf("resultStr: %s, err: %+v", resultStr, err)
				}
				common.SaveAgentNodeInfoToBackend(newItem)
			}
		}()
	}
}


func (agentWatch *AgentWatcher) InitAgentWatcher() {
	go agentWatch.GenerateAgentWatcherLoop()
	go agentWatch.ConcurrencyWatchAgentWatcherQueue()

}

