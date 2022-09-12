package process

import (
	"op-agent/config"
	"os"
	"sort"
	"strings"

	"github.com/openark/golib/log"
	"op-agent/util"
)

var ThisHostname	string
var ThisHostIp		string
var ThisHostToken	string

func init() {
	GetHostNameAndIp([]string{})
}

func GetHostNameAndIp(filter []string) {
	var (
		err error
		ipsList []string
	)

	ThisHostname, err = os.Hostname()
	if err != nil {
		log.Errorf("Cannot resolve self hostname; required. Aborting. %+v", err)
	}
	ipsList, err = util.GetLocalIP(filter)
	if err != nil {
		log.Errorf("Cannot GetLocalIP . %+v", err)
	}
	sort.Strings(ipsList)
	ThisHostIp = strings.Join(ipsList, ",")
}

func InitHostName() {
	filterIPSegment, err := GetNonLiveIpSegment()
	if err != nil {
		log.Errorf("GetNonLiveIpSegment err: %+v", err)
	}
	GetHostNameAndIp(filterIPSegment)
}

func InitToken() {
	var (
		tokenFile 	string
		token		string
	)
	tokenFile = config.Config.TokenFilePath
	if !util.FileExists(tokenFile) {
		token = generateTokenAndWriteFile(tokenFile)
	} else {
		if result, err := util.ReadFileToString(tokenFile); err != nil {
			log.Errorf("Get token through ReadFileToString %s err %+v. Will generated a new token.", tokenFile, err)
			token = generateTokenAndWriteFile(tokenFile)
		} else {
			if result == "" {
				token = generateTokenAndWriteFile(tokenFile)
			} else {
				token = result
			}
		}
	}
	ThisHostToken = token
}

func generateTokenAndWriteFile(filePath string) string {
	token := util.ProcessToken.Hash
	if err := util.WriteFile(filePath, token); err != nil {
		log.Errorf("Write token %s to file %s err %+v, different tokens will be generated when the agent starts",token, filePath, err)
	}
	return token
}