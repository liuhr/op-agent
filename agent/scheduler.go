package agent

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/openark/golib/log"

	"op-agent/config"
	"op-agent/process"
	"op-agent/util"
)


//task scheduling
type Scheduler struct {
	jobEventChan chan *JobEvent
	jobPlanTable map[string]*JobSchedulerPlan
	jobExecutingTable map[string]*JobExecuteInfo
	jobResultChan chan *JobExecuteResult
}

var (
	G_scheduler *Scheduler
)

func (scheduler *Scheduler) handleJobEvent(jobEvent *JobEvent) {
	var (
		jobSchedulePlan *JobSchedulerPlan
		jobExisted bool
		err error
	)

	switch jobEvent.EventType {
	case JOB_EVENT_SAVE:
		if jobEvent.Job.OnceJob == 1 {
			if _, jobExecuting := scheduler.jobExecutingTable[jobEvent.Job.JobName]; jobExecuting {
				log.Warningf("%s (onceJob=%v) is already in the running queue, skipping next loop.", jobEvent.Job.JobName, jobEvent.Job.OnceJob)
				return
			}
			jobSchedulePlan = &JobSchedulerPlan{
				Job: jobEvent.Job,
			}
			scheduler.TryStartJob(jobSchedulePlan)
			return
		}
		if jobEvent.Job.CronExpr == "" {
			return
		}
		if jobSchedulePlan, err = BuildJobSchedulePlan(jobEvent.Job); err != nil {
			log.Errorf("BuildJobSchedulePlan of %s Failed: %s",jobEvent.Job.JobName, err.Error())
			return
		}
		scheduler.jobPlanTable[jobEvent.Job.JobName] = jobSchedulePlan
	case JOB_EVENT_DELETE:
		if jobSchedulePlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.JobName]; jobExisted {
			log.Infof("Delete the task of %s from  jobPlanTable", jobEvent.Job.JobName)
			delete(scheduler.jobPlanTable, jobEvent.Job.JobName)
		}
	case JOB_EVENT_KILL:
		if jobExecuteInfo, jobExecuting := scheduler.jobExecutingTable[jobEvent.Job.JobName]; jobExecuting {
			log.Infof("Will kill job %s", jobExecuteInfo.Job.Command)
			jobExecuteInfo.CancelFunc()
		}
	}

}

//Run task
func (scheduler *Scheduler) TryStartJob(jobPlan *JobSchedulerPlan) {
	var (
		jobExecuteInfo *JobExecuteInfo
		jobExecuting bool
	)

	if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobPlan.Job.JobName]; jobExecuting {
		log.Warningf("%s (onceJob=%v) is still running, skipping the loop.", jobPlan.Job.JobName, jobPlan.Job.OnceJob)
		return
	}

	jobExecuteInfo = BuildJobExecuteInfo(jobPlan)
	scheduler.jobExecutingTable[jobPlan.Job.JobName] = jobExecuteInfo

	log.Infof("Perform task: %s (onceJob=%v) planTime: %v realTime: %v", jobExecuteInfo.Job.JobName,jobExecuteInfo.Job.OnceJob,jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	G_executor.ExecutorJob(jobExecuteInfo)
}


//Recalculate task scheduling status
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		jobPlan *JobSchedulerPlan
		now time.Time
		nearTime *time.Time
	)
	if len(scheduler.jobPlanTable) == 0 {
		scheduleAfter = 1 * time.Second
		return
	}

	now = time.Now()
	for _, jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			scheduler.TryStartJob(jobPlan)
			jobPlan.NextTime = jobPlan.Expr.Next(now) //Update next execution time
		}
		//Count the latest task time to be expired
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}

	//Next schedule interval (most recently scheduled task - current time)
	scheduleAfter = (*nearTime).Sub(now)
	return
}

//Processing task results
func (scheduler *Scheduler) handleJobResult (result *JobExecuteResult) {
	var (
		jobLog *JobLog
	)
	//Generate operation log
	jobLog =  &JobLog{
		HostName: process.ThisHostname,
		HostIP: process.ThisHostIp,
		Token: process.ThisHostToken,
		JobName: result.ExecuteInfo.Job.JobName,
		Command: result.ExecuteInfo.Job.Command,
		Version: result.ExecuteInfo.Job.Version,
		OnceJob: result.ExecuteInfo.Job.OnceJob,
		Output: string(result.Output),
		PlanTime: result.ExecuteInfo.PlanTime.Format("2006-01-02 15:04:05"),
		ScheduleTime: result.ExecuteInfo.RealTime.Format("2006-01-02 15:04:05"),
		StartTime: result.StartTime.Format("2006-01-02 15:04:05"),
		EndTime: result.EndTime.Format("2006-01-02 15:04:05"),
	}

	resultLogDir, err := util.MakeDir(config.Config.ResultLogDir)
	if err != nil {
		resultLogDir = "/tmp"
	}
	//resultLogFile := resultLogDir+"/"+result.ExecuteInfo.Job.JobName+fmt.Sprintf("-%d",time.Now().Unix()) + ".log" //  ./log/test.py-1627469221_log
	resultLogFile := resultLogDir+"/"+result.ExecuteInfo.Job.JobName + ".log" //  ./log/test.py.log

	go func(logFile string) {
		if err := ioutil.WriteFile(resultLogFile, result.Output,0644); err != nil {
			log.Errorf("Exception writing task(%s) execution result to file(%s): %s", result.ExecuteInfo.Job.Command, logFile, err.Error())
		}
	}(resultLogFile)

	if strings.Count(jobLog.Output, "\n") >= config.Config.JobResultLines {
		log.Warningf("Task(%s) execution result lines is more than %d, it will not be saved to meta and a local log will be generated", result.ExecuteInfo.Job.Command,config.Config.JobResultLines)
		jobLog.Output = fmt.Sprintf("The number of execution result lines is too large. The execution result is saved at: %s", resultLogFile)
	}

	if result.Err != nil {
		jobLog.Err = result.Err.Error()
	} else {
		jobLog.Err = ""
	}

	if result.ExecuteInfo.Job.OnceJob == 1 {
		go func() {
			dat := map[string]string{"status": "2",
				"token": jobLog.Token,
				"version": jobLog.Version,
				"jobname": jobLog.JobName,
				"hostname": jobLog.HostName,
				"host": jobLog.HostIP,
				"plantime": jobLog.PlanTime,
				"scheduletime": jobLog.ScheduleTime,
				"starttime": jobLog.StartTime,
				"endtime": jobLog.EndTime,
				"output": jobLog.Output,
				"errinfo": jobLog.Err,
				"saveOnceJobLogFlag": "1",
				"saveStatusFlag": "1",
			}

			if err := UpdateOnceJobStatus(dat); err == nil {
				delete(scheduler.jobExecutingTable, result.ExecuteInfo.Job.JobName)
			} else {
				UpdateOnceJobStatusToMeta(dat)
				if err := SaveOnceJobLog(jobLog); err == nil {
					delete(scheduler.jobExecutingTable, result.ExecuteInfo.Job.JobName)
				}
			}
			/*if err := SaveOnceJobLog(jobLog); err == nil {
				delete(scheduler.jobExecutingTable, result.ExecuteInfo.Job.JobName)
			}*/
		}()
	} else {
		delete(scheduler.jobExecutingTable, result.ExecuteInfo.Job.JobName)
	}

	G_logSink.Append(jobLog)
	log.Infof("Task %s execution info: cmd=%s, Version=%s, OnceJob=%d, PlanTime=%s, ScheduleTime=%s, StartTime=%s, EndTime=%s",
		jobLog.JobName, jobLog.Command, jobLog.Version, jobLog.OnceJob, jobLog.PlanTime, jobLog.ScheduleTime, jobLog.StartTime, jobLog.EndTime)
	//fmt.Println("执行结果：", string(result.Output), result.Err)
}


//Scheduling coroutine
func (scheduler *Scheduler) schedulerLoop() {
	var (
		jobEvent *JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult *JobExecuteResult
	)
	scheduleAfter = scheduler.TrySchedule()
	scheduleTimer = time.NewTimer(scheduleAfter)

	for {
		select {
		case jobEvent = <- scheduler.jobEventChan: //Monitor task change events
			scheduler.handleJobEvent(jobEvent)
		case <- scheduleTimer.C:
		case jobResult = <- scheduler.jobResultChan:
			scheduler.handleJobResult(jobResult)
		}
		//Schedule task
		scheduleAfter = scheduler.TrySchedule()
		scheduleTimer.Reset(scheduleAfter)
	}
}

//Push task change events
func (scheduler *Scheduler) PushJobEvent(jobEvent *JobEvent) {
	scheduler.jobEventChan <- jobEvent
}

//Initialize scheduler
func InitScheduler() (err error){
	G_scheduler = &Scheduler {
		jobEventChan: make(chan *JobEvent, 100),
		jobPlanTable: make(map[string]*JobSchedulerPlan),
		jobExecutingTable: make(map[string]*JobExecuteInfo),
		jobResultChan: make(chan *JobExecuteResult, 1000),
	}
	go G_scheduler.schedulerLoop()
	return
}

//Return task execution result
func (scheduler *Scheduler) PushJobResult(jobResult *JobExecuteResult) {
	scheduler.jobResultChan <- jobResult
}
