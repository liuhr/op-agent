package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"

	"github.com/asmcos/requests"
	"github.com/openark/golib/log"
	"github.com/openark/golib/sqlutils"

	"op-agent/config"
	"op-agent/db"
	"op-agent/process"
	"op-agent/util"
)

type Controller struct {}

var (
	G_controller *Controller
)

func (controler *Controller) initJobs() error {
	var (
		results []map[string]string
		err error
	)
	if results,err = controler.ListAllJobs(); err != nil {
		log.Errorf("job_control take all jobs err :%v", err)
	}
	InitJobMap(results)
	for _, jobContent := range GjobsMap {
		if controler.CheckBlackIps(jobContent) {
			log.Warningf("Found local ip in the Black List. Will not execute %s", jobContent.JobName)
			continue
		}
		if jobContent.OnceJob == 1 {
			if err := controler.getOnceJobVersion(jobContent); err != nil {
				log.Errorf("getOnceJobVersion of %s err %+v", jobContent.JobName, err)
				continue
			}
			if jobContent.Version == "" {
				log.Warningf("Found onceJob %s(%s) has been executed, or is not in the white list, will be ignored.", jobContent.JobName, jobContent.Command)
				continue
			}
		}
		jobEvent := &JobEvent{
			EventType: JOB_EVENT_SAVE,
			Job: jobContent,
		}
		//Pass it to scheduler
		G_scheduler.PushJobEvent(jobEvent)
	}
	return nil
}

func (controler *Controller) UpdateChangedJobs(jobs []map[string]string) error {
	if jobs == nil {
		return nil
	}
	differenceJobs, purgeJobs := CompareJobs(jobs)
	for _, jobContent := range differenceJobs {
		jobEvent := &JobEvent{}
		if controler.CheckBlackIps(jobContent) {
			log.Warningf("Found local ip in the Black List. Will not execute %s", jobContent.JobName)
			jobEvent.EventType = JOB_EVENT_DELETE
		} else {
			jobEvent.EventType = JOB_EVENT_SAVE
		}
		if jobContent.OnceJob == 1 {
			if jobContent.Version == "" {
				jobEvent.EventType = JOB_EVENT_DELETE
			}
		}

		jobEvent.Job = jobContent
		//Pass it to scheduler
		G_scheduler.PushJobEvent(jobEvent)
	}
	for _, jobContent := range purgeJobs {
		jobEvent := &JobEvent{
			EventType: JOB_EVENT_DELETE,
			Job: jobContent,
		}
		//Pass it to scheduler
		G_scheduler.PushJobEvent(jobEvent)
	}
	return nil
}


type RequestData struct {
	Code  		string    `json:"Code"`
	Message   	string    `json:"Message"`
	Details		map[string]string `json: "Details"`
}

// The function is upload Heartbeat and get changed jobs and new packages through API
func (controler *Controller) continuesHandleRequests() (timeTick time.Duration) {
	var (
		requestData 		RequestData
		handleOpAgentApi	string
		resp				*requests.Response
		err					error
	)


	defer func() {
		timeTick = time.Duration(rand.Intn(config.Config.ContinuesDiscoverWithInSeconds)) * time.Second
		if requestData.Details == nil {
			return
		}
		if value, ok := requestData.Details["agent_request_server_time_range"]; ok {
			//The value of rand.Intn()  cannot be 0
			timeTick = time.Duration(rand.Intn(util.ConvStrToInt(value)+1)) * time.Second
			return
		}
		return
	}()

	SinceLastReceiveDataFromServerDuration := SinceLastReceiveDataFromServer()
	if SinceLastReceiveDataFromServerDuration.Seconds() < 300 {  // 5 minutes
		return
	}

	data := requests.Datas{
		"hostname": process.ThisHostname,
		"host": process.ThisHostIp,
		"token": process.ThisHostToken,
		"port": strings.Replace(config.Config.ListenAddress, ":", "", -1),
		"appVersion": config.NewAppVersion(),
	}

	for i := 1; i <= 3; i++ {
		req := requests.Requests()
		req.SetTimeout(5)
		myServer := util.TakeRandServerHost(config.Config.OpManagers)
		if myServer == "" {
			log.Errorf("Pls check config.Config.OpServers. Found null.")
			return
		}
		handleOpAgentApi = fmt.Sprintf("http://%s:%d/api/handle-op-agent",myServer, config.Config.OpManagerPort)
		log.Infof("api: %s postData: %+v",handleOpAgentApi, data)
		resp, err = req.PostJson(handleOpAgentApi, requests.Auth{config.Config.OpManagerUser, config.Config.OpManagerPass}, data)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Errorf("req.PostJson err %+v", err)
		return
	}
	resultStr := resp.Text()
	if !strings.Contains(resultStr, "OK") {
		log.Errorf("Request %s err %s", handleOpAgentApi, resultStr)
		return
	}

	if err = json.Unmarshal([]byte(resultStr), &requestData); err != nil {
		log.Errorf("Unmarshal resultStr err: %+v", err)
		return
	}

	writeNodeStatus := requestData.Details["writeNodeStatus"]
	if writeNodeStatus != "" {
		errInfo, _ := base64.StdEncoding.DecodeString(writeNodeStatus)
		log.Errorf("writeNodeStatus err: %s", string(errInfo))
	}

	getAllJobsStatus := requestData.Details["GetAllJobsStatus"]
	if getAllJobsStatus != "" {
		errInfo, _ := base64.StdEncoding.DecodeString(getAllJobsStatus)
		log.Errorf("GetAllJobsStatus err: %s", string(errInfo))
	} else {
		var data []map[string]string
		jobsListStr := requestData.Details["allJobs"]
		if err := json.Unmarshal([]byte(jobsListStr), &data); err != nil {
			log.Errorf("Unmarshal jobsListStr err: %+v", err)
		} else {
			go controler.UpdateChangedJobs(data)
		}
	}

	getPackagesTaskStatus := requestData.Details["GetPackagesTaskStatus"]
	if getPackagesTaskStatus != "" {
		errInfo, _ := base64.StdEncoding.DecodeString(getPackagesTaskStatus)
		log.Errorf("GetPackagesTaskStatus err: %s", string(errInfo))
	} else {
		var data map[string]string
		packageTaskMapStr := requestData.Details["PackageTask"]
		if err := json.Unmarshal([]byte(packageTaskMapStr), &data); err != nil {
			log.Errorf("Unmarshal packageTaskMapStr err: %+v", err)
		} else {
			//go WatchPackageVersion(data)
		}
	}

	if nonLiveIPSeg, ok := requestData.Details["nonliveips"]; ok {
		if nonLiveIPSeg != "" {
			go process.GetHostNameAndIp(strings.Split(nonLiveIPSeg, ","))
		}
	}
	atomic.StoreInt64(&LastReceiveDataFromServerUnixNano, time.Now().UnixNano())
	return
}

func (controler *Controller) getOnceJobVersion(job *Job) error {
	query := `
			select 
				version 
			from 
				oncejobtask 
			where ip='%s' and jobname='%s' and hasrun = '0' order by add_time limit 1`
	query = fmt.Sprintf(query, process.ThisHostIp, job.JobName)
	err := db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		job.Version = m.GetString("version")
		return nil
	})
	return err
}

func (controler *Controller) watchKiller() {
	go func() {
		var (
			results []map[string]string
			err error
		)
		registrationTick := time.Tick(time.Duration(rand.Intn(3) + 2) * time.Second)
		for range registrationTick {
			if results, err = controler.ListKillJobs(); err != nil {
				continue
			}
			jobKillMap := InitJobKillMap(results)
			for _, jobContent := range jobKillMap {
				jobEvent := &JobEvent{
					EventType: JOB_EVENT_KILL,
					Job: jobContent,
				}
				G_scheduler.PushJobEvent(jobEvent)
			}
		}
	}()
}

func (controler *Controller) ListKillJobs() ([]map[string]string,error) {
	var (
		query string
		err error
		results []map[string]string
	)
	results = make([]map[string]string,0)

	query = fmt.Sprintf("select * from jobs where enabled = 1 and killFlag=1")

	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		resultMap := map[string]string{}
		resultMap["jobname"] = m.GetString("jobname")
		results = append(results, resultMap)
		return nil
	})
	return results,err
}

func (controler *Controller) ListAllJobs() ([]map[string]string,error) {
	var (
		query string
		err error
		results []map[string]string
	)
	results = make([]map[string]string,0)
	query = fmt.Sprintf("select * from jobs where enabled = 1")

	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		resultMap := map[string]string{}
		resultMap["jobname"] = m.GetString("jobname")
		resultMap["command"] = m.GetString("command")
		resultMap["cronexpr"] = m.GetString("cronexpr")
		resultMap["oncejob"] = m.GetString("oncejob")
		resultMap["timeout"] = m.GetString("timeout")
		resultMap["synflag"] = m.GetString("synflag")
		resultMap["whiteips"] = m.GetString("whiteips")
		resultMap["blackips"] = m.GetString("blackips")
		resultMap["killFlag"] = m.GetString("killFlag")
		resultMap["cpushares"] = m.GetString("cpushares")
		resultMap["cpuquotaus"] = m.GetString("cpuquotaus")
		resultMap["memorylimit"] = m.GetString("memorylimit")
		resultMap["memoryswlimit"] = m.GetString("memoryswlimit")
		resultMap["ioreadlimit"] = m.GetString("ioreadlimit")
		resultMap["iowritelimit"] = m.GetString("iowritelimit")
		resultMap["iolimitdevice"] = m.GetString("iolimitdevice")
		results = append(results, resultMap)
		return nil
	})
	return results,err
}

func (controler *Controller) CheckBlackIps(job *Job) bool {
	var (
		findInWhite bool
		whiteList string
	)
	whiteList  = strings.Replace(job.WhiteIps, " ", "",-1)

	localIpsList := strings.Split(process.ThisHostIp, ",")
	for _, ip := range localIpsList {
		if strings.Contains(job.BlackIps, ip) {
			return true
		}
		if  whiteList != "" {
			if strings.Contains(job.WhiteIps, ip) {
				findInWhite = true
			}
		}
	}
	if whiteList != "" {
		if !findInWhite {
			return true
		}
	}
	return false
}

func (controler *Controller) ContinuesDiscover() {
	var (
		timeTick time.Duration
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
	)
	atomic.StoreInt64(&LastReceiveDataFromServerUnixNano, time.Now().UnixNano())
	timeTick = time.Duration(rand.Intn(config.Config.ContinuesDiscoverWithInSeconds)) * time.Second
	scheduleTimer = time.NewTimer(timeTick)
	for {
		select {
		case <- scheduleTimer.C:
		}
		scheduleAfter = G_controller.continuesHandleRequests()
		if scheduleAfter == time.Duration(0) * time.Second {
			scheduleAfter = timeTick
		}
		scheduleTimer.Reset(scheduleAfter)
	}
}


func UpdateChangedJobs (jobs []map[string]string) error {
	return G_controller.UpdateChangedJobs(jobs)
}

func InitJobControl() {
	G_controller.initJobs()
	go G_controller.ContinuesDiscover()
	//G_controller.watchKiller()
}
