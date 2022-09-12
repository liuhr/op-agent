package agent

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/asmcos/requests"
	"github.com/openark/golib/log"

	"op-agent/config"
	"op-agent/db"
	"op-agent/process"
	"op-agent/util"
)

type LogSink struct {
	logChan chan *JobLog
	autoCommitChan chan *LogBatch
}


var (
	G_logSink *LogSink
)

func (logSink *LogSink) saveLogs(bach *LogBatch)  {
	/*sqlStr := `insert into joblogs
				    (hostname, ip, jobname, command, version, plantime, scheduletime, starttime, endtime, output, err)
				values `
	values := []interface{}{}

	for _, logInfo := range bach.Logs {
		sqlStr += "(?, ?, ?, ?, ?, ?, ?, ?, ?,?, ?),"
		values = append(values, logInfo.HostName, logInfo.HostIP, logInfo.JobName, logInfo.Command, logInfo.Version, logInfo.PlanTime, logInfo.ScheduleTime,
			logInfo.StartTime, logInfo.EndTime, logInfo.Output, logInfo.Err)
	}
	sqlStr = strings.TrimRight(sqlStr, ",")
	_, err := db.ExecDb(sqlStr, values...)
	if err != nil {
		log.Errorf("Run saveLogs to meta failed: %s", err.Error())
	}*/

	for _, logInfo := range bach.Logs {
		if err := WriteJobLogsThroughServerApi(logInfo); err != nil {
			log.Errorf("WriteJobLogsThroughServerApi err %+v", err)
		}
	}
}

func WriteJobLogsThroughServerApi(jobLog *JobLog) error{
	var (
		resp 				*requests.Response
		err 				error
		saveJobExecLogApi	string
	)

	data := requests.Datas{
		"hostname": jobLog.HostName,
		"token": process.ThisHostToken,
		"host": jobLog.HostIP,
		"jobname": strings.Replace(jobLog.JobName, " ", "", -1),
		"plantime": jobLog.PlanTime,
		"scheduletime": jobLog.ScheduleTime,
		"starttime": jobLog.StartTime,
		"endtime": jobLog.EndTime,
		"output": base64.StdEncoding.EncodeToString([]byte(jobLog.Output)),
		"errinfo": base64.StdEncoding.EncodeToString([]byte(jobLog.Err)),
	}

	for i := 1; i <= 3; i++ {
		req := requests.Requests()
		req.SetTimeout(5)
		opManager := util.TakeRandServerHost(config.Config.OpManagers)
		if opManager == "" {
			log.Errorf("Pls check config.Config.OpManagers. Found null.")
			return err
		}
		saveJobExecLogApi = fmt.Sprintf("http://%s:%d/api/save-job-execute-log",myServer, config.Config.OpManagerPort)
		log.Infof(saveJobExecLogApi)
		resp, err = req.PostJson(saveJobExecLogApi, requests.Auth{config.Config.OpManagerUser, config.Config.OpManagerPass}, data)

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


func (logSink *LogSink) writeLoop() {
	var (
		logInfo *JobLog
		logBatch *LogBatch
		commitTimer *time.Timer
		timeoutBatch *LogBatch
	)
	for {
		select {
		case logInfo = <- logSink.logChan:
			if logBatch == nil {
				logBatch = &LogBatch{}
				commitTimer = time.AfterFunc(
					time.Duration(config.Config.JobLogCommitTimeOut) * time.Millisecond,
					func(batch *LogBatch) func() {
						return func() {
							logSink.autoCommitChan <- batch
						}
					}(logBatch),
				)
			}
			logBatch.Logs = append(logBatch.Logs, logInfo)
			if len(logBatch.Logs) >= config.Config.JobLogBatchSize {
				logSink.saveLogs(logBatch)
				logBatch = nil
				commitTimer.Stop()
			}
		case timeoutBatch = <- logSink.autoCommitChan:
			if timeoutBatch != logBatch {
				continue
			}
			logSink.saveLogs(timeoutBatch)
			logBatch = nil
		}
	}
}

func InitLogSink () (err error) {
	G_logSink = &LogSink{
		logChan: make(chan *JobLog, 1000),
		autoCommitChan: make(chan *LogBatch, 1000),
	}
	go G_logSink.writeLoop()
	return
}

func (logSink *LogSink) Append(jobLog *JobLog) {
	select {
	case logSink.logChan <- jobLog:
	default:
		// 队列满了就丢弃
	}
}



func SaveOnceJobLog(logInfo *JobLog) error {
	sqlResult, err := db.ExecDb(`
				update 
					oncejobtask
				set hasrun='1', starttime=?, endtime=?, output=?, err=? 
				where token=? and version=? and jobname=?
			`,
		logInfo.StartTime, logInfo.EndTime, logInfo.Output,logInfo.Err, logInfo.Token, logInfo.Version, logInfo.JobName,
	)
	if err != nil {
		log.Errorf("Save onceJob %s(%s) result log to meta err: %+v", logInfo.JobName, logInfo.Command, err)
		return err
	}
	rows, err := sqlResult.RowsAffected()
	if err != nil {
		log.Errorf("Failed to calculate the number of logs (%s: %s) written to the backend. %+v", logInfo.JobName, logInfo.Command, err)
		return err
	}
	if rows < 1 {
		errInfo := fmt.Sprintf("Save onceJob %s(%s) result log to meta did not take effect.", logInfo.JobName, logInfo.Command)
		log.Errorf(errInfo)
		return errors.New(errInfo)
	}
	return nil
}
