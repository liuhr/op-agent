package http

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/openark/golib/log"
	"io/ioutil"
	"net/http"
	"op-agent/agent"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/auth"
	"github.com/martini-contrib/render"

	"op-agent/agentCli"
	"op-agent/config"
	"op-agent/process"
	oraft "op-agent/raft"
	"op-agent/util"
)

var apiSynonyms = map[string]string{}

// APIResponseCode is an OK/ERROR response code
type APIResponseCode int

var registeredPaths = []string{}

const (
	ERROR APIResponseCode = iota
	OK
)

type HttpAPI struct {
	URLPrefix string
}

var API HttpAPI = HttpAPI{}

func (this *APIResponseCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.String())
}

func (this *APIResponseCode) String() string {
	switch *this {
	case ERROR:
		return "ERROR"
	case OK:
		return "OK"
	}
	return "unknown"
}

// HttpStatus returns the respective HTTP status for this response
func (this *APIResponseCode) HttpStatus() int {
	switch *this {
	case ERROR:
		return http.StatusInternalServerError
	case OK:
		return http.StatusOK
	}
	return http.StatusNotImplemented
}

// APIResponse is a response returned as JSON to various requests.
type APIResponse struct {
	Code    APIResponseCode
	Message string
	Details interface{}
}

func Respond(r render.Render, apiResponse *APIResponse) {
	r.JSON(apiResponse.Code.HttpStatus(), apiResponse)
}

// A configurable endpoint that can be for regular status checks or whatever.  While similar to
// Health() this returns 500 on failure.  This will prevent issues for those that have come to
// expect a 200
// It might be a good idea to deprecate the current Health() behavior and roll this in at some
// point
func (this *HttpAPI) StatusCheck(params martini.Params, r render.Render, req *http.Request) {
	health, err := process.HealthTest()
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: fmt.Sprintf("Application node is unhealthy %+v", err), Details: health})
		return
	}
	Respond(r, &APIResponse{Code: OK, Message: fmt.Sprintf("Application node is healthy"), Details: health})
}

func (this *HttpAPI) registerSingleAPIRequest(m *martini.ClassicMartini, path string, handler martini.Handler, allowProxy bool) {
	registeredPaths = append(registeredPaths, path)
	fullPath := fmt.Sprintf("%s/api/%s", this.URLPrefix, path)

	//if allowProxy && config.Config.RaftEnabled {
	//	m.Get(fullPath, raftReverseProxy, handler)
	//} else {
	m.Get(fullPath, handler)
	//}
}

func (this *HttpAPI) getSynonymPath(path string) (synonymPath string) {
	pathBase := strings.Split(path, "/")[0]
	if synonym, ok := apiSynonyms[pathBase]; ok {
		synonymPath = fmt.Sprintf("%s%s", synonym, path[len(pathBase):])
	}
	return synonymPath
}

func (this *HttpAPI) registerAPIRequestInternal(m *martini.ClassicMartini, path string, handler martini.Handler, allowProxy bool) {
	this.registerSingleAPIRequest(m, path, handler, allowProxy)

	if synonym := this.getSynonymPath(path); synonym != "" {
		this.registerSingleAPIRequest(m, synonym, handler, allowProxy)
	}
}

func (this *HttpAPI) registerAPIRequestNoProxy(m *martini.ClassicMartini, path string, handler martini.Handler) {
	this.registerAPIRequestInternal(m, path, handler, false)
}

func (this *HttpAPI) GetAppVersion(params martini.Params, r render.Render, req *http.Request) {
	version := config.NewAppVersion()
	if version != "" {
		r.JSON(200, &APIResponse{Code: OK, Details: version})
		return
	}
	Respond(r, &APIResponse{Code: ERROR, Message: "can not find version"})
	return
}

func (this *HttpAPI) registerAPIRequest(m *martini.ClassicMartini, path string, handler martini.Handler) {
	this.registerAPIRequestInternal(m, path, handler, true)
}

func (this *HttpAPI) JobSave(params martini.Params, r render.Render, req *http.Request) {
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	dataString := string(body)
	dataStringList := strings.Split(dataString, "&")

	var dat map[string]string
	err = json.Unmarshal([]byte(dataStringList[0]), &dat)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error() + " " + "Unmarshal params failed"})
		return
	}
	err = agentCli.SaveJob(dat)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	var result map[string]string
	if result, err = agentCli.ListOneJob(dat["jobname"]); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, &APIResponse{Code: OK, Details: result})
	return
}

func (this *HttpAPI) SaveJobLog(params martini.Params, r render.Render, req *http.Request){
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}

	dataString := string(body)
	dataStringList := strings.Split(dataString, "&")
	if len(dataStringList) == 0 {
		r.JSON(500, &APIResponse{Code: ERROR, Message: "Params can not be null"})
		return
	}

	var dat map[string]string
	err = json.Unmarshal([]byte(dataStringList[0]), &dat)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error() + " " + "Unmarshal params failed"})
		return
	}
	err = agentCli.WriteJobExecLogs(dat)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, &APIResponse{Code: OK, Message: ""})
	return
}

func (this *HttpAPI) SaveOnceJobStatusOrLog(params martini.Params, r render.Render, req *http.Request) {
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}

	dataString := string(body)
	dataStringList := strings.Split(dataString, "&")
	if len(dataStringList) == 0 {
		r.JSON(500, &APIResponse{Code: ERROR, Message: "Params can not be null"})
		return
	}

	var dat map[string]string
	err = json.Unmarshal([]byte(dataStringList[0]), &dat)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error() + " " + "Unmarshal params failed"})
		return
	}

	err = agentCli.SaveOnceJobStatusOrLog(dat)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, &APIResponse{Code: OK, Message: ""})
	return
	
	r.JSON(200, &APIResponse{Code: OK, Message: ""})
	return
}


func (this *HttpAPI) NodeStatusSave(params martini.Params, r render.Render, req *http.Request) {
	defer req.Body.Close()
	req.Header.Set("Content-Type","application/json")
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	dataString := string(body)
	dataStringList := strings.Split(dataString, "&")
	if len(dataStringList) == 0 {
		r.JSON(500, &APIResponse{Code: ERROR, Message: "Params can not be null"})
		return
	}

	var dat map[string]string
	err = json.Unmarshal([]byte(dataStringList[0]), &dat)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error() + " " + "Unmarshal params failed"})
		return
	}
	err = agentCli.WriteNodeStatus(dat)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, &APIResponse{Code: OK, Message: ""})
	return
}


func (this *HttpAPI) GetAllJobs(params martini.Params, r render.Render, req *http.Request) {
	jobsResults, err := agentCli.GetAllJobs("")
	if err != nil {
		r.JSON(200, &APIResponse{Code: ERROR, Details: err})
		return
	}
	Respond(r, &APIResponse{Code: OK, Message: "", Details: jobsResults})
	return
}

func (this *HttpAPI) GetActiveAgents(params martini.Params, r render.Render, req *http.Request) {
	activeAgents, err := agentCli.GetAllActiveHosts()
	if err != nil {
		r.JSON(200, &APIResponse{Code: ERROR, Details: err})
		return
	}
	Respond(r, &APIResponse{Code: OK, Message: "", Details: activeAgents})
	return
}

func (this *HttpAPI) UpdatePluginVersion(params martini.Params, r render.Render, req *http.Request) {
	data := map[string]string{}
	err := agent.WatchPluginVersion(data)
	if err != nil {
		r.JSON(200, &APIResponse{Code: ERROR, Details: err})
		return
	}
	Respond(r, &APIResponse{Code: OK, Message: ""})
	return
}

func (this *HttpAPI) DataReceiveFromManger(params martini.Params, r render.Render, req *http.Request) {
	defer req.Body.Close()
	req.Header.Set("Content-Type","application/json")
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	dataString := string(body)
	dataStringList := strings.Split(dataString, "&")
	if len(dataStringList) == 0 {
		r.JSON(500, &APIResponse{Code: ERROR, Message: "Params can not be null"})
		return
	}
	var (
		getJobsInfo string
		getPluginsInfo string
		receiveData map[string]string
		respondData map[string]string
	)
	respondData = map[string]string{}
	err = json.Unmarshal([]byte(dataStringList[0]), &receiveData)
	if activeRaftManagers, ok := receiveData["raftAvailableNodes"]; ok {
		if activeRaftManagers != "" {
			log.Infof("Found activeRaftManagers: %s, will update config.Config.OpManagers", activeRaftManagers)
			config.Config.OpManagers = strings.Split(activeRaftManagers, ",")
		}
	}

	getAllJobsStatus := receiveData["GetAllJobsStatus"]
	if getAllJobsStatus != "" {
		errInfo, _ := base64.StdEncoding.DecodeString(getAllJobsStatus)
		log.Errorf("GetAllJobsStatus err: %s", string(errInfo))
	} else {
		var data []map[string]string
		jobsMapListStr := receiveData["allJobs"]
		if jobsMapListStr != "" {
			if err := json.Unmarshal([]byte(jobsMapListStr), &data); err != nil {
				getJobsInfo = fmt.Sprintf("jobsMapListStr: %s Unmarshal jobsMapListStr err: %+v", jobsMapListStr,err)
				log.Errorf(getJobsInfo)
			} else {
				go agent.UpdateChangedJobs(data)
			}
		}
	}

	getPluginsTaskStatus := receiveData["GetPackagesTaskStatus"]
	if getPluginsTaskStatus != "" {
		errInfo, _ := base64.StdEncoding.DecodeString(getPluginsTaskStatus)
		log.Errorf("getPluginsTaskStatus err: %s", string(errInfo))
	} else {
		var data map[string]string
		pluginTaskMapStr := receiveData["PackageTask"]
		if pluginTaskMapStr != "" {
			if err := json.Unmarshal([]byte(pluginTaskMapStr), &data); err != nil {
				getPluginsInfo =  fmt.Sprintf("packageTaskMapStr:%s Unmarshal pluginTaskMapStr err: %+v", pluginTaskMapStr, err)
				log.Errorf(getPluginsInfo)
			} else {
				go agent.WatchPluginVersion(data)
			}
		}
	}

	if nonLiveIPSeg, ok := receiveData["nonliveips"]; ok {
		if nonLiveIPSeg != "" {
			go process.GetHostNameAndIp(strings.Split(nonLiveIPSeg, ","))
		}
	}

	respondData["app_version"] = config.NewAppVersion()
	respondData["hostname"] = process.ThisHostname
	respondData["ip"] = process.ThisHostIp
	respondData["token"] = process.ThisHostToken
	respondData["getPackagesInfo"] = getPluginsInfo
	respondData["getJobsInfo"] = getJobsInfo
	//SinceLastReceiveDataFromServerDuration := base.SinceLastReceiveDataFromServer()
	atomic.StoreInt64(&agent.LastReceiveDataFromServerUnixNano, time.Now().UnixNano())
	Respond(r, &APIResponse{Code: OK, Details: respondData})
	return
}


func (this *HttpAPI) GetOnceJobVersion(params martini.Params, r render.Render, req *http.Request) {
	host := strings.Replace(params["ip"]," ","",-1)
	jobName := strings.Replace(params["jobName"]," ","",-1)
	version, err := agentCli.GetOnceJobVersion(host, jobName)
	if err != nil {
		Respond(r, &APIResponse{Code: ERROR, Message: fmt.Sprintf("%+v", err)})
		return
	}
	Respond(r, &APIResponse{Code: OK, Message: "", Details: version})
	return
}

func (this *HttpAPI) HandleOpAgent(params martini.Params, r render.Render, req *http.Request) {
	defer req.Body.Close()
	req.Header.Set("Content-Type","application/json")
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	dataString := string(body)
	dataStringList := strings.Split(dataString, "&")
	if len(dataStringList) == 0 {
		r.JSON(500, &APIResponse{Code: ERROR, Message: "Params can not be null"})
		return
	}

	var (
		dat map[string]string
		packageTask map[string]string
		allJobs []map[string]string
	)
	results := make(map[string]string)
	err = json.Unmarshal([]byte(dataStringList[0]), &dat)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error() + " " + "Unmarshal params failed"})
		return
	}
	//Save Node Status To backend db.
	err = agentCli.WriteNodeStatus(dat)
	if err != nil {
		results["writeNodeStatus"] = base64.StdEncoding.EncodeToString([]byte(err.Error()))
	} else {
		results["writeNodeStatus"] = ""
	}

	//Get packages change task
	packageTask, err = agentCli.GetPackagesTask(dat["token"])
	if err != nil {
		results["GetPackagesTaskStatus"] = base64.StdEncoding.EncodeToString([]byte(err.Error()))
	} else {
		results["GetPackagesTaskStatus"] = ""
	}
	result, _ := json.Marshal(packageTask)
	results["PackageTask"] = string(result)

	//Get all jobs
	allJobs, err = agentCli.GetAllJobs(dat["token"])
	if err != nil {
		results["GetAllJobsStatus"] = base64.StdEncoding.EncodeToString([]byte(err.Error()))
	} else {
		results["GetAllJobsStatus"] = ""
	}
	result, _ = json.Marshal(allJobs)
	results["allJobs"] = string(result)

	//Get variables
	if err = agentCli.GetVariables(results); err != nil {
		results["getVariablesStatus"] = base64.StdEncoding.EncodeToString([]byte(err.Error()))
	}

	r.JSON(200, &APIResponse{Code: OK, Message: "", Details: results})
	return
}


func (this *HttpAPI) CommonRequest(params martini.Params, r render.Render, req *http.Request) {
	var findFlag bool
	var script string
	var param string
	var outputFlag string
	process := config.Config.Processes
	if len(process) < 1 {
		err := fmt.Errorf("scripts from process is null")
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error(), Details: ""})
		return
	}
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error(), Details: ""})
		return
	}
	initInfo := string(body)
	initInfoList := strings.Split(initInfo, "&")
	if len(initInfoList) == 0 {
		err := fmt.Errorf("parameter can not be null")
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error(), Details: ""})
		return
	}
	for _, v := range initInfoList {
		var dat map[string]string
		err := json.Unmarshal([]byte(v), &dat)
		if err != nil {
			r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error() + " " + "Unmarshal params failed", Details: ""})
			return
		}
		for _, vmap := range process {
			if v, ok := dat["key"]; ok {
				if vmap["key"] == v {
					script = vmap["script"]
					param = vmap["param"]
					outputFlag = vmap["outputFlag"]
					findFlag = true
					break
				}
			} else {
				r.JSON(500, &APIResponse{Code: ERROR, Message: "comman api must add 'key' param", Details: ""})
				return
			}

		}

		if !findFlag {
			continue
		}

		for _, prm := range strings.Split(param, ",") {
			if v, ok := dat[prm]; ok {
				script = strings.Replace(script, fmt.Sprintf("{%s}", prm), v, -1)
			}
		}

		if len(script) == 0 {
			continue
		}
		if outputFlag == "1" {
			row, err := util.RunCommandOutput(script)
			if err != nil {
				r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error(), Details: ""})
				return
			}
			r.JSON(200, &APIResponse{Code: OK, Message: "", Details: row})
			return
		} else {
			err := util.RunCommandNoOutput(script)
			if err != nil {
				r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error(), Details: ""})
				return
			}
			r.JSON(200, &APIResponse{Code: OK, Message: "", Details: ""})
			return
		}
	}

	if !findFlag || len(script) == 0 {
		r.JSON(500, &APIResponse{Code: ERROR, Message: "find no script to run", Details: ""})
		return
	}
	r.JSON(500, &APIResponse{Code: ERROR, Message:"", Details: fmt.Sprintf("script :%s outputFlag:%s", script, outputFlag)})
	return
}

// RaftFollowerHealthReport is initiated by followers to report their identity and health to the raft leader.
func (this *HttpAPI) RaftFollowerHealthReport(params martini.Params, r render.Render, req *http.Request, user auth.User) {
	if !oraft.IsRaftEnabled() {
		Respond(r, &APIResponse{Code: ERROR, Message: "raft-state: not running with raft setup"})
		return
	}
	err := oraft.OnHealthReport(params["authenticationToken"], params["raftBind"], params["raftAdvertise"])
	if err != nil {
		Respond(r, &APIResponse{Code: ERROR, Message: fmt.Sprintf("Cannot create snapshot: %+v", err)})
		return
	}
	r.JSON(http.StatusOK, "health reported")
}

// RegisterRequests makes for the de-facto list of known API calls
func (this *HttpAPI) RegisterRequests(m *martini.ClassicMartini) {
	var apiEndpoint string
	this.registerAPIRequestNoProxy(m, "raft-follower-health-report/:authenticationToken/:raftBind/:raftAdvertise", this.RaftFollowerHealthReport)
	this.registerAPIRequest(m, "version", this.GetAppVersion)
	if config.Config.ApiEndpoint != "" {
		apiEndpoint = config.Config.ApiEndpoint
	} else {
		apiEndpoint = config.DefaultApiEndpoint
	}
	m.Post(apiEndpoint, this.CommonRequest)
	//op-manager side
	m.Post("/api/job-save", this.JobSave)
	m.Post("/api/save-job-execute-log", this.SaveJobLog)
	m.Post("/api/save-node-status", this.NodeStatusSave)
	m.Post("/api/handle-op-agent", this.HandleOpAgent)
	m.Post("/api/save-onceJob-statusOrExecuteLog", this.SaveOnceJobStatusOrLog)
	m.Get("/api/get-jobs", this.GetAllJobs)
	m.Get("/api/get-once-job-version/:ip/:jobName", this.GetOnceJobVersion)
	m.Get("/api/active_agent", this.GetActiveAgents)
	//op-agent side
	m.Get("/api/update-agent-plugin", this.UpdatePluginVersion)
	m.Post("/api/dataReceive", this.DataReceiveFromManger)
	
	// Configurable status check endpoint
	if config.Config.StatusEndpoint == config.DefaultStatusAPIEndpoint {
		this.registerAPIRequestNoProxy(m, "status", this.StatusCheck)
	} else {
		m.Get(config.Config.StatusEndpoint, this.StatusCheck)
	}
}
