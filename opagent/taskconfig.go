package opagent

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/openark/golib/log"
)

type TaskConfiguration struct {
	JobName								string
	Command								string
	CronExpr							string
	OnceJob								uint
	Timeout								uint
	SynFlag								uint

	CPUShares							int
	CPUQuotaUs							int
	MemoryLimit							int
	MemorySwLimit						int
	IOReadLimit							int
	IOWriteLimit						int
	IOLimitDevice						string

	Enabled								uint

	WhiteHosts							[]string
	BlackHosts							[]string
}

var TaskConfig = newTaskConfiguration()
var readFileNames []string

func newTaskConfiguration() *TaskConfiguration {
	return &TaskConfiguration{
		OnceJob:			0,
		Timeout:			0,
		SynFlag:			0,
		CPUShares:			128,
		Enabled:			1,
		WhiteHosts:			[]string{},
		BlackHosts:			[]string{},
	}
}

// ForceRead reads configuration from given file name or bails out if it fails
func ForceRead(fileName string) *TaskConfiguration {
	_, err := read(fileName)
	if err != nil {
		log.Fatal("Cannot read config file:", fileName, err)
	}
	readFileNames = []string{fileName}
	return TaskConfig
}

func Read(fileNames ...string) *TaskConfiguration {
	for _, fileName := range fileNames {
		read(fileName)
	}
	readFileNames = fileNames
	return TaskConfig
}

func read(fileName string) (*TaskConfiguration, error) {
	if fileName == "" {
		return TaskConfig, fmt.Errorf("config file param must's null")
	}
	file, err := os.Open(fileName)
	if err != nil {
		return TaskConfig, err
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(TaskConfig)
	if err == nil {
		log.Infof("Read config: %s", fileName)
	} else {
		log.Fatal("Cannot read config file:", fileName, err)
	}
	return TaskConfig, err
}