package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/asmcos/requests"
	"github.com/gorhill/cronexpr"
	"github.com/openark/golib/log"

	"op-agent/config"
	"op-agent/db"
	"op-agent/util"
)

const (
	JOB_EVENT_SAVE = 1
	JOB_EVENT_DELETE = 0
	JOB_EVENT_KILL = 2
	CgroupCmd = "/usr/bin/cgexec"
	ProcsFile = "cgroup.procs"
	CPUShareFile = "cpu.shares"
	CPUQuotaUsFile = "cpu.cfs_quota_us"
	MemoryLimitFile = "memory.limit_in_bytes"
	MemorySmLimitFile = "memory.memsw.limit_in_bytes"
	IOReadlimitFile = "blkio.throttle.read_bps_device"
	IOWriteLimitFile = "blkio.throttle.write_bps_device"
	CGroupMemRootPath = "/sys/fs/cgroup/memory/"
	CGroupCPURootPath = "/sys/fs/cgroup/cpu/"
	CGroupBlkioRootPath = "/sys/fs/cgroup/blkio/"
)

var	GjobsMap map[string]*Job = make(map[string]*Job)
var LastReceiveDataFromServerUnixNano int64

type Job struct {
	JobName string
	Command string
	CronExpr string
	OnceJob uint
	Version string
	Timeout uint
	SynFlag uint
	WhiteIps string
	BlackIps string
	CpuShares int
	CpuQuotaUs int
	Memorylimit int
	Memoryswlimit int
	IoReadlimit int
	IoWritelimit int
	Iolimitdevice string
	Enabled uint
	Md5Sum string
	FromApi uint
}


type JobEvent struct {
	EventType int
	Job *Job
}


type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo
	Output []byte
	Err error
	StartTime time.Time
	EndTime time.Time
}

type JobLog struct {
	HostName string
	Token	string
	HostIP	string
	JobName string
	Command string
	Version string
	OnceJob uint
	Err string
	Output string
	PlanTime string
	ScheduleTime string
	StartTime string
	EndTime string
}


type LogBatch struct {
	Logs []*JobLog
}

type JobSchedulerPlan struct {
	Job *Job
	Expr *cronexpr.Expression //parsed cronexpr expression
	NextTime time.Time
}

type JobExecuteInfo struct {
	Job *Job
	PlanTime time.Time
	RealTime time.Time
	CancelCtx context.Context
	CancelFunc context.CancelFunc
}

type RequestInfo struct {
	Hostname	string
	Host 		string
	Port		string
	Jobs		[]map[string]string
	Packages	[]map[string]string
}

type FromApiRespond struct {
	Code		string
	Message		string
	Details		map[string]string
}

func SinceLastReceiveDataFromServer() time.Duration {
	timeNano := atomic.LoadInt64(&LastReceiveDataFromServerUnixNano)
	if timeNano == 0 {
		return 0
	}
	return time.Since(time.Unix(0, timeNano))
}

func InitJobMap(results []map[string]string) {
	for _, row := range results {
		rowString, _ := json.Marshal(row)
		job := FillJobOb(row)
		job.Md5Sum = util.Md5(string(rowString))
		GjobsMap[row["jobname"]] = job
	}
}

func InitJobKillMap(results []map[string]string) map[string]*Job {
	var (
		jobKillMap map[string]*Job
	)
	jobKillMap = make(map[string]*Job)
	for _, row := range results {
		jobKillMap[row["jobname"]] = &Job{
			JobName: row["jobname"],
		}
	}
	return jobKillMap

}

func CompareJobs(results []map[string]string) (map[string]*Job, map[string]*Job) {
	var (
		jobMap map[string]*Job
		purgeMap map[string]*Job
		md5Sum string
		diffFlag bool
	)
	jobMap = make(map[string]*Job)
	purgeMap = make(map[string]*Job)

	//Capture new tasks and changed tasks
	for _, row := range results {
		diffFlag = false
		rowString, _ := json.Marshal(row)
		md5Sum = util.Md5(string(rowString))
		if _, ok := GjobsMap[row["jobname"]]; ok {
			if GjobsMap[row["jobname"]].Md5Sum != md5Sum {
				diffFlag = true
			} else {
				if v, ok := row["oncejob"]; ok {
					if v == "1" {
						diffFlag = true
					}
				}
			}
		} else {
			diffFlag = true
		}

		if diffFlag {
			job := FillJobOb(row)
			job.Md5Sum = md5Sum
			jobMap[row["jobname"]] = job
			GjobsMap[row["jobname"]] = jobMap[row["jobname"]]
		}
	}

	//Capture the deleted tasks.
	for jobName,content := range GjobsMap {
		purgeFlag := true
		for _, result := range results {
			if jobName == result["jobname"] {
				purgeFlag = false
			}
		}
		if purgeFlag {
			purgeMap[jobName] = content
		}
	}

	//Delete the deleted task from the global variable GjobsMap.
	for jobName, _ := range purgeMap {
		delete(GjobsMap,jobName)
	}

	return jobMap, purgeMap
}


func FillJobOb(row map[string]string) *Job {
	job := &Job{}
	if len(row) == 0 {
		return job
	}
	job = &Job{
		JobName: row["jobname"],
		Command: row["command"],
		CronExpr: row["cronexpr"],
		OnceJob: util.ConvStrToUInt(row["oncejob"]),
		Version: row["oncejobversion"],
		Timeout: util.ConvStrToUInt(row["timeout"]),
		SynFlag: util.ConvStrToUInt(row["synflag"]),
		WhiteIps: row["whiteips"],
		BlackIps: row["blackips"],
		CpuShares: util.ConvStrToInt(row["cpushares"]),
		CpuQuotaUs: util.ConvStrToInt(row["cpuquotaus"]),
		Memorylimit: util.ConvStrToInt(row["memorylimit"]),
		Memoryswlimit: util.ConvStrToInt(row["memoryswlimit"]),
		IoReadlimit: util.ConvStrToInt(row["ioreadlimit"]),
		IoWritelimit: util.ConvStrToInt(row["iowritelimit"]),
		Iolimitdevice: row["iolimitdevice"],
		Enabled: util.ConvStrToUInt(row["enabled"]),
	}
	return job
}


//Construct task execution plan
func BuildJobSchedulePlan(job *Job) (jobSchedulePlan *JobSchedulerPlan, err error) {
	var (
		expr *cronexpr.Expression
	)
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return
	}

	jobSchedulePlan = &JobSchedulerPlan{
		Job: job,
		Expr: expr,
		NextTime: expr.Next(time.Now()),
	}
	return
}

func BuildJobExecuteInfo(jobSchedulePlan *JobSchedulerPlan) (jobExecuteInfo *JobExecuteInfo) {
	if jobSchedulePlan.Expr == nil {
		jobExecuteInfo = &JobExecuteInfo{
			Job: jobSchedulePlan.Job,
			PlanTime: time.Now(),
			RealTime: time.Now(),
		}
	} else {
		jobExecuteInfo = &JobExecuteInfo{
			Job:      jobSchedulePlan.Job,
			PlanTime: jobSchedulePlan.NextTime,
			RealTime: time.Now(),
		}
	}
	jobExecuteInfo.CancelCtx, jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())
	return
}

func UpdateOnceJobStatus(dat map[string]string)  error {
	if err := UpdateOnceJobStatusOrLogThroughApi(dat); err != nil {
		log.Errorf("UpdateOnceJobStatusOrLogThroughApi err %+v", err)
		return err
		//UpdateOnceJobStatusToMeta(dat)
	}
	return nil
}

func UpdateOnceJobStatusToMeta(dat map[string]string) {
	update :=  "update oncejobtask set status= ? where token=? and version=? and jobname=?"
	if _ , err := db.ExecDb(update, dat["status"], dat["token"], dat["version"], dat["jobname"]); err != nil {
		log.Errorf("%s params (%s, %s, %s, %s) err %+v", update, dat["status"], dat["token"], dat["version"], dat["jobname"], err)
		return
	}
	return
}

func UpdateOnceJobStatusOrLogThroughApi(dat map[string]string) error {
	var (
		resp 				*requests.Response
		err 				error
		saveJobStatusOrExecLogApi	string
	)
	data := requests.Datas{
		"hostname": dat["hostname"],
		"token": dat["token"],
		"version": dat["version"],
		"host": dat["host"],
		"jobname": dat["jobname"],
		"plantime": dat["plantime"],
		"scheduletime": dat["scheduletime"],
		"starttime": dat["starttime"],
		"endtime": dat["endtime"],
		"output": dat["output"],
		"errinfo": dat["errinfo"],
		"status": dat["status"],
		"saveOnceJobLogFlag": dat["saveOnceJobLogFlag"],
		"saveStatusFlag": dat["saveStatusFlag"],
	}
	for i := 1; i <= 3; i++ {
		req := requests.Requests()
		req.SetTimeout(5)
		myServer := util.TakeRandServerHost(config.Config.OpManagers)
		if myServer == "" {
			log.Errorf("Pls check config.Config.OpManagers. Found null.")
			return err
		}
		saveJobStatusOrExecLogApi = fmt.Sprintf("http://%s:%d/api/save-onceJob-statusOrExecuteLog",myServer, config.Config.OpManagerPort)
		log.Infof(saveJobStatusOrExecLogApi)
		resp, err = req.PostJson(saveJobStatusOrExecLogApi, requests.Auth{config.Config.OpManagerUser, config.Config.OpManagerPass}, data)

		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return err
	}
	resultStr := resp.Text()
	if strings.Contains(resultStr, "OK") {
		return nil
	} else {
		return errors.New(resultStr)
	}
	return nil

}


type JobsData struct {
	Code  		string    `json:"Code"`
	Message   	string    `json:"Message"`
	Details		[]map[string]string `json: "Details"`
}

func GetJobsFromServerApi() ([]map[string]string,error){
	var (
		data JobsData
		results []map[string]string
	)
	results = make([]map[string]string,0)
	req := requests.Requests()
	resp, err := req.Get(fmt.Sprintf("http://%s:%d/api/get-jobs",config.Config.OpManagers, config.Config.OpManagerPort),requests.Auth{config.Config.OpManagerUser, config.Config.OpManagerPass})
	if err != nil {
		return results, err
	}
	resultStr := resp.Text()
	if strings.Contains(resultStr, "OK") {
		if err := json.Unmarshal([]byte(resultStr), &data); err == nil {
			results = data.Details
			return results, nil
		} else {
			return results, err
		}
	} else {
		return results, errors.New(resultStr)
	}
	return results, nil
}

type OnceJobData struct {
	Code  		string    `json:"Code"`
	Message   	string    `json:"Message"`
	Details		string	  `json: "Details"`
}
func GetOnceJobVersionFromServerApi(host string, jobName string) (string, error){
	var (
		result string
		data OnceJobData
	)
	req := requests.Requests()
	resp, err := req.Get(fmt.Sprintf("http://%s:%d/api/get-once-job-version/%s/%s",config.Config.OpManagers, config.Config.OpManagerPort, host, jobName),requests.Auth{config.Config.OpManagerUser, config.Config.OpManagerPass})
	if err != nil {
		return result, err
	}
	resultStr := resp.Text()
	if strings.Contains(resultStr, "OK") {
		if err := json.Unmarshal([]byte(resultStr), &data); err == nil {
			result = data.Details
			return result, nil
		} else {
			return result, err
		}
	} else {
		return result, errors.New(resultStr)
	}
	return result, nil
}
