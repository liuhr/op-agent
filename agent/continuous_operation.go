package agent

import (
	"time"

	"github.com/openark/golib/log"
)

func ContinuousOperation() {
	log.Infof("continuous operation: setting up")
	//Start log process
	InitLogSink()

	//Start job executor
	InitExecutor()

	//Start job scheduler
	InitScheduler()

	//Start job controller
	//InitJobControl()

	loopTimer := time.NewTimer(1 * time.Second)
	log.Infof("continuous operation: starting")
	for {
		select {
		case <-loopTimer.C:
		}
	}
}
