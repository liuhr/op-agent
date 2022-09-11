package process

import (
	"os"
	"strings"

	"github.com/openark/golib/log"
	"op-agent/util"
)

var ThisHostname string
var ThisHostIp string

func init() {
	var (
		err error
		ipsList []string
	)

	ThisHostname, err = os.Hostname()
	if err != nil {
		log.Errorf("Cannot resolve self hostname; required. Aborting. %+v", err)
	}
	ipsList, err = util.GetLocalIP()
	if err != nil {
		log.Errorf("Cannot GetLocalIP . %+v", err)
	}
	ThisHostIp = strings.Join(ipsList, ",")
}
