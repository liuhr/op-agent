package manager

import (
	"encoding/json"
	"github.com/openark/golib/log"
	"op-agent/raft"
)

// AsyncRequest represents an entry in the async_request table
type CommandApplier struct {
}

func NewCommandApplier() *CommandApplier {
	applier := &CommandApplier{}
	return applier
}

func (applier *CommandApplier) ApplyCommand(op string, value []byte) interface{} {
	switch op {
	case "heartbeat":
		return nil
	case "leader-uri":
		return applier.leaderURI(value)
	case "request-health-report":
		return applier.healthReport(value)
	}
	return log.Errorf("Unknown command op: %s", op)
}

func (applier *CommandApplier) leaderURI(value []byte) interface{} {
	var uri string
	if err := json.Unmarshal(value, &uri); err != nil {
		return log.Errore(err)
	}
	oraft.LeaderURI.Set(uri)
	return nil
}

func (applier *CommandApplier) healthReport(value []byte) interface{} {
	var authenticationToken string
	if err := json.Unmarshal(value, &authenticationToken); err != nil {
		return log.Errore(err)
	}
	oraft.ReportToRaftLeader(authenticationToken)
	return nil
}
