package agent

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"op-agent/process"
	"op-agent/util"

	"github.com/openark/golib/log"
)


type Executor struct {}

var (
	G_executor *Executor
)

func (executor *Executor) ExecutorJob(jobExecuteInfo *JobExecuteInfo) (result *JobExecuteResult) {
	result = &JobExecuteResult{}
	if jobExecuteInfo.Job.OnceJob == 1 {
		dat := map[string]string{"saveStatusFlag": "1",
			"status":"1",
			"token": process.ThisHostToken,
			"version": jobExecuteInfo.Job.Version,
			"jobname": jobExecuteInfo.Job.JobName}
		if err := UpdateOnceJobStatus(dat); err != nil {
			UpdateOnceJobStatusToMeta(dat)
		}
	}
	if jobExecuteInfo.Job.SynFlag == 0 {
		go func() {
			executor.executorJobNow(jobExecuteInfo)
		}()
	} else {
		result = executor.executorJobNow(jobExecuteInfo)
	}
	return result
}

func (executor *Executor) executorJobNow(jobExecuteInfo *JobExecuteInfo) (result *JobExecuteResult) {
	var (
		cmd *exec.Cmd
		err error
		runTimeoutSec uint
		ctx context.Context
		cancel context.CancelFunc
		//cgroupProcsFileList []string
	)

	result = &JobExecuteResult{
		ExecuteInfo: jobExecuteInfo,
		Output: make([]byte, 0),
	}
	result.StartTime = time.Now()

	params := []string{"-c",jobExecuteInfo.Job.Command}
	cmd = exec.Command("bash", params...)
	log.Infof("Command: %+v",cmd)
	buf := bytes.Buffer{}
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if jobExecuteInfo.Job.Timeout == 0 {
		runTimeoutSec = 31536000 // one year
	} else {
		runTimeoutSec = jobExecuteInfo.Job.Timeout
	}
	ctx, cancel = context.WithTimeout(context.Background(),
		time.Duration(runTimeoutSec)*time.Second)
	defer cancel()

	waitChan := make(chan struct{}, 1)
	defer close(waitChan)
	go func() {
		select {
		case <-ctx.Done():
			log.Errorf("Timeout, kill job %s pid:%d",jobExecuteInfo.Job.Command, cmd.Process.Pid)
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		case <-waitChan:
			return
		}
	}()

	if err = cmd.Start(); err != nil {
		log.Errorf("cmd.Start() err %+v", err)
		goto End
	}
	//cgroupProcsFileList = executor.InitJobCGroup(jobExecuteInfo.Job)
	//for _, procsFile := range cgroupProcsFileList {
	//	log.Infof("Write pid(%d) of %s to %s", cmd.Process.Pid, jobExecuteInfo.Job.Command, procsFile)
	//	if err := util.WriteFile(procsFile, fmt.Sprintf("%d", cmd.Process.Pid)); err != nil {
	//		log.Errorf("Write pid %d to %s err %+v", cmd.Process.Pid ,procsFile, err)
	//	}
	//}
	if err = cmd.Wait(); err != nil {
		log.Errorf("cmd.Wait() err %+v %s", err, string(buf.Bytes()))
		goto End
	}
End:
	result.EndTime = time.Now()
	result.Output = buf.Bytes()
	result.Err = err
	G_scheduler.PushJobResult(result)
	waitChan <- struct{}{}
	return
}


func (executor *Executor) InitJobCGroup(job *Job) (procsFileList []string) {
	procsFileList = make([]string,0)
	if exist := util.FileExists(CgroupCmd); !exist {
		log.Errorf("InitJobCGroup failed %s is not existed", CgroupCmd)
		return
	}
	if job.CpuShares != 0 || job.CpuQuotaUs != 0 {
		if procFile := executor.InitCPUCgroup(job); procFile != "" {
			procsFileList = append(procsFileList, procFile)
		}
	}
	if job.Memorylimit != 0 || job.Memoryswlimit != 0 {
		if procFile := executor.InitMemCgroup(job); procFile != "" {
			procsFileList = append(procsFileList, procFile)
		}
	}
	if job.IoReadlimit != 0 || job.IoWritelimit != 0 {
		if procFile := executor.InitIOCgroup(job); procFile != "" {
			procsFileList = append(procsFileList, procFile)
		}
	}
	return
}


func (executor *Executor) InitCPUCgroup(job *Job) (procFile string){
	var (
		err error
	)
	if exist := util.PathExists(CGroupCPURootPath); exist {
		deleteCmd := "cgdelete -g cpu:/" + job.JobName
		if err = util.RunCommandNoOutput(deleteCmd); err != nil {
			log.Errorf("run %s err %+v", deleteCmd, err)
		}
		cpuCGroupRoot := CGroupCPURootPath + job.JobName
		// /sys/fs/cgroup/cpu/$JobName
		if _, err = util.MakeDir(cpuCGroupRoot); err != nil {
			log.Errorf("InitJobCGroup failed MakeDir(%s) err %s+v", cpuCGroupRoot, err)
			return
		}

		cpuSharesFile := cpuCGroupRoot + "/" + CPUShareFile
		// /sys/fs/cgroup/cpu/$JobName/cpu.shares
		cpuQuotaUsFile := cpuCGroupRoot + "/" + CPUQuotaUsFile
		// /sys/fs/cgroup/cpu/$JobName/cpu.cfs_quota_us

		if job.CpuShares != 0 {
			if err = util.WriteFile(cpuSharesFile, fmt.Sprintf("%d", job.CpuShares)); err != nil {
				log.Errorf("WriteFile %s err %+v", cpuSharesFile, err)
				return
			}
		}

		if job.CpuQuotaUs != 0 {
			if err = util.WriteFile(cpuQuotaUsFile, fmt.Sprintf("%d", job.CpuQuotaUs)); err != nil {
				log.Errorf("WriteFile %s err %+v", cpuQuotaUsFile, err)
				return
			}
		}

		procFile = cpuCGroupRoot + "/" + ProcsFile
		// /sys/fs/cgroup/cpu/$JobName/cgroup.procs
	}
	return
}

func (executor *Executor) InitMemCgroup(job *Job) (procFile string){
	var (
		err error
	)
	if exist := util.PathExists(CGroupMemRootPath); exist {
		deleteCmd := "cgdelete -g memory:/" + job.JobName
		if err = util.RunCommandNoOutput(deleteCmd); err != nil {
			log.Errorf("run %s err %+v", deleteCmd, err)
		}
		memCGroupRoot := CGroupMemRootPath + job.JobName
		// /sys/fs/cgroup/memory/$JobName
		if _, err = util.MakeDir(memCGroupRoot); err != nil {
			log.Errorf("InitJobCGroup failed MakeDir(%s) err %s+v", memCGroupRoot, err)
			return
		}

		memoryLimitFile := memCGroupRoot + "/" + MemoryLimitFile
		// /sys/fs/cgroup/memory/$JobName/memory.limit_in_bytes
		memorySWLimitFile := memCGroupRoot + "/" + MemorySmLimitFile
		// /sys/fs/cgroup/memory/$JobName/memory.memsw.limit_in_bytes

		if job.Memorylimit != 0  {
			if err = util.WriteFile(memoryLimitFile, fmt.Sprintf("%d", job.Memorylimit)); err != nil {
				log.Errorf("WriteFile %s err %+v", memoryLimitFile, err)
				return
			}
		}
		if job.Memoryswlimit != 0 {
			if err = util.WriteFile(memorySWLimitFile, fmt.Sprintf("%d", job.Memoryswlimit)); err != nil {
				log.Errorf("WriteFile %s err %+v", memorySWLimitFile, err)
				return
			}
		}
		procFile = memCGroupRoot + "/" + ProcsFile
		// /sys/fs/cgroup/memory/$JobName/cgroup.procs
	}
	return
}

func (executor *Executor) InitIOCgroup(job *Job) (procFile string){
	var (
		err error
		deviceId string
	)
	if exist := util.PathExists(CGroupBlkioRootPath); exist {
		deleteCmd := "cgdelete -g blkio:/" + job.JobName
		if err = util.RunCommandNoOutput(deleteCmd); err != nil {
			log.Errorf("run %s err %+v", deleteCmd, err)
		}

		ioCGroupRoot := CGroupBlkioRootPath + job.JobName
		// /sys/fs/cgroup/blkio/$JobName
		if _, err = util.MakeDir(ioCGroupRoot); err != nil {
			log.Errorf("InitJobCGroup failed MakeDir(%s) err %s+v", ioCGroupRoot, err)
			return
		}
		if job.Iolimitdevice == "" {
			log.Errorf("Found no IO device to setup IO cgroup.")
			return
		}
		shellCmd := "ls -l " + job.Iolimitdevice + "| awk '{print $5,$6}' | sed 's/ //g' | tr ',' ':'"
		// ls -l /dev/sdb | awk '{print $5,$6}' | sed 's/ //g' | tr ',' ':'
		deviceId, err = util.RunCommandOutput(shellCmd)
		if err != nil {
			log.Errorf("Run %s err %+v",shellCmd, err)
			return
		}
		if deviceId == "" {
			log.Errorf("Found no IO device")
			return
		}

		ioReadLimitFile := ioCGroupRoot + "/" + IOReadlimitFile
		// /sys/fs/cgroup/blkio/$JobName/blkio.throttle.read_bps_device
		ioWriteLimitFile := ioCGroupRoot + "/" + IOWriteLimitFile
		// /sys/fs/cgroup/blkio/$JobName/blkio.throttle.write_bps_device

		if job.IoReadlimit != 0 {
			readLimit := fmt.Sprintf("%s %d", deviceId, job.IoReadlimit)
			if err = util.WriteFile(ioReadLimitFile, readLimit); err != nil {
				log.Errorf("WriteFile %s err %+v", ioReadLimitFile, err)
				return
			}
		}

		if job.IoWritelimit != 0 {
			writeLimit := fmt.Sprintf("%s %d", deviceId, job.IoWritelimit)
			if err = util.WriteFile(ioWriteLimitFile, writeLimit); err != nil {
				log.Errorf("WriteFile %s err %+v", ioWriteLimitFile, err)
				return
			}
		}

		procFile = ioCGroupRoot + "/" + ProcsFile
		// /sys/fs/cgroup/blkio/$JobName/cgroup.procs
	}
	return
}

func InitExecutor() (err error) {
	G_executor = &Executor{}
	return
}
