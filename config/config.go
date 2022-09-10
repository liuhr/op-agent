package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/openark/golib/log"
)

const (
	DefaultStatusAPIEndpoint = "/api/status"
	DefaultApiEndpoint       = "/api/commonRequest"
	HealthPollSeconds        = 1
	GetPackageSeconds      = 2
	WatchPackageTaskStatusSeconds      = 2
	ActiveNodeExpireSeconds  = 5
	RaftHealthPollSeconds    = 10
	DiscoveryQueueCapacity   = 100000
)

var configurationLoaded chan bool = make(chan bool)

func NewAppVersion() string {
	AppVersion := "2.0.0"
	return AppVersion
}


// Configuration makes for  configuration input, which can be provided by user via JSON formatted file.
// Some of the parameteres have reasonable default values, and some (like database credentials) are
// strictly expected from user.
type Configuration struct {
	Debug                               bool   // set debug mode (similar to --debug option)
	ListenAddress                       string // Where this system HTTP should listen for TCP
	RaftEnabled                         bool   // When true, setup this system in a raft consensus layout. When false (default) all Raft* variables are ignored
	RaftBind                            string
	RaftDataDir                         string
	RaftAdvertise                       string
	DefaultRaftPort                     int      // if a RaftNodes entry does not specify port, use this one
	RaftNodes                           []string // Raft nodes to make initial connection with
	EtcdEndpoints                       []string
	EtcdDailTimeout                     uint
	RaftNodesStatusCheckIntervalSeconds uint
	RaftNodesStatusAlertProcess         string
	RaftLeaderDomain                    string
	DomainCheckIntervalSeconds          uint
	//SwithDomainProcess                     string
	SwithDomainProcess                       []string
	HTTPAdvertise                            string   // optional, for raft setups, what is the HTTP address this node will advertise to its peers (potentially use where behind NAT or when rerouting ports; example: "http://11.22.33.44:3030")
	InstancePollSeconds                      uint     // Number of seconds between instance reads
	UnseenInstanceForgetHours                uint     // Number of hours after which an unseen instance is forgotten
	SnapshotTopologiesIntervalHours          uint     // Interval in hour between snapshot-topologies invocation. Default: 0 (disabled)
	InstanceBulkOperationsWaitTimeoutSeconds uint     // Time to wait on a single instance when doing bulk (many instances) operation
	AuthenticationMethod                     string   // Type of autherntication to use, if any. "" for none, "basic" for BasicAuth, "multi" for advanced BasicAuth, "proxy" for forwarded credentials via reverse proxy, "token" for token based access
	HTTPAuthUser                             string   // Username for HTTP Basic authentication (blank disables authentication)
	HTTPAuthPassword                         string   // Password for HTTP Basic authentication
	AuthUserHeader                           string   // HTTP header indicating auth user, when AuthenticationMethod is "proxy"
	PowerAuthUsers                           []string // On AuthenticationMethod == "proxy", list of users that can make changes. All others are read-only.
	ServeAgentsHttp                          bool     // Spawn another HTTP interface dedicated for orchestrator-agent
	AgentsUseSSL                             bool     // When "true" this system will listen on agents port with SSL as well as connect to agents via SSL
	UseSSL                                   bool     // Use SSL on the server web port
	UseMutualTLS                             bool     // When "true" Use mutual TLS for the server's web and API connections
	SSLSkipVerify                            bool     // When using SSL, should we ignore SSL certification error
	SSLPrivateKeyFile                        string   // Name of SSL private key file, applies only when UseSSL = true
	SSLCertFile                              string   // Name of SSL certification file, applies only when UseSSL = true
	SSLCAFile                                string   // Name of the Certificate Authority file, applies only when UseSSL = true
	SSLValidOUs                              []string // Valid organizational units when using mutual TLS
	StatusEndpoint                           string   // Override the status endpoint.  Defaults to '/api/status'
	ApiEndpoint                              string   // Defaults to '/api/commonRequest'
	StatusOUVerify                           bool     // If true, try to verify OUs when Mutual TLS is on.  Defaults to false
	URLPrefix                                string   // URL prefix to run this system on non-root web path, e.g.  to put it behind nginx.

	MySQLReadTimeoutSeconds        uint
	MySQLConnectTimeoutSeconds     uint
	MySQLRejectReadOnly            bool // Reject read only connections https://github.com/go-sql-driver/mysql#rejectreadonly
	MySQLMaxPoolConnections        int  // The maximum size of the connection pool to the Orchestrator backend.
	MySQLConnectionLifetimeSeconds int  // Number of seconds the mysql driver will keep database connection alive before recycling it

	ConnBackendDbFlag bool
	BackendDbHosts    string
	BackendDbPort     uint
	BackendDbUser     string
	BackendDbPass     string
	BackendDb         string

	AgentNodesPullPackagesIntervalSeconds	uint
	DealWithAgentsDescMaxConcurrency		uint
	AgentDownloadPackageMaxConcurrency		uint
	DiscoverOpAgentIntervalLists			[]int
	DiscoverOpAgentConcurrency				uint

	LastSeenTimeoutSeconds					uint

	NumberOfJobLogDaysKeepPerHost			uint
	NumberOfJobLogDaysKeepInactiveHost		uint

	OpAgentUser								string
	OpAgentPass								string
	OpAgentPort								int
	OpAgentApiEndpoint						string
	OpAgentDataReceiveApiEndPoint			string

	Processes []map[string]string
}

// Config is *the* configuration instance, used globally to get configuration data
var Config = newConfiguration()
var readFileNames []string

func newConfiguration() *Configuration {
	return &Configuration{
		Debug:                                    false,
		ListenAddress:                            ":3000",
		HTTPAdvertise:                            "",
		StatusEndpoint:                           DefaultStatusAPIEndpoint,
		StatusOUVerify:                           false,
		RaftBind:                                 "127.0.0.1:10008",
		RaftDataDir:                              "",
		DefaultRaftPort:                          10008,
		RaftNodes:                                []string{},
		EtcdEndpoints:                            []string{},
		EtcdDailTimeout:                          5,
		RaftNodesStatusCheckIntervalSeconds:      60,
		RaftLeaderDomain:                         "",
		DomainCheckIntervalSeconds:               60,
		SwithDomainProcess:                       []string{},
		InstancePollSeconds:                      5,
		UnseenInstanceForgetHours:                240,
		SnapshotTopologiesIntervalHours:          0,
		InstanceBulkOperationsWaitTimeoutSeconds: 10,
		AuthenticationMethod:                     "",
		HTTPAuthUser:                             "",
		HTTPAuthPassword:                         "",
		AuthUserHeader:                           "X-Forwarded-User",
		PowerAuthUsers:                           []string{"*"},
		UseSSL:                                   false,
		UseMutualTLS:                             false,
		SSLValidOUs:                              []string{},
		SSLSkipVerify:                            false,
		SSLPrivateKeyFile:                        "",
		SSLCertFile:                              "",
		SSLCAFile:                                "",
		URLPrefix:                                "",
		MySQLReadTimeoutSeconds:                  30,
		MySQLConnectTimeoutSeconds:               5,
		MySQLRejectReadOnly:                      false,
		MySQLMaxPoolConnections:                  1000, // limit concurrent conns to backend DB
		MySQLConnectionLifetimeSeconds:           0,
		AgentNodesPullPackagesIntervalSeconds:	  120,
		DealWithAgentsDescMaxConcurrency:		  50,
		AgentDownloadPackageMaxConcurrency:		  10,
		NumberOfJobLogDaysKeepPerHost:			  3,
		NumberOfJobLogDaysKeepInactiveHost:		  30,
		LastSeenTimeoutSeconds:					  600,
		Processes:                                []map[string]string{},
		ConnBackendDbFlag:                        false,
		DiscoverOpAgentIntervalLists:			  []int{120, 240, 300, 360, 420, 480, 540},
		DiscoverOpAgentConcurrency:				  100,
	}
}

func (this *Configuration) postReadAdjustments() error {
	if this.URLPrefix != "" {
		// Ensure the prefix starts with "/" and has no trailing one.
		this.URLPrefix = strings.TrimLeft(this.URLPrefix, "/")
		this.URLPrefix = strings.TrimRight(this.URLPrefix, "/")
		this.URLPrefix = "/" + this.URLPrefix
	}
	if this.RaftEnabled && this.RaftDataDir == "" {
		return fmt.Errorf("RaftDataDir must be defined since raft is enabled (RaftEnabled)")
	}
	if this.RaftEnabled && this.RaftBind == "" {
		return fmt.Errorf("RaftBind must be defined since raft is enabled (RaftEnabled)")
	}
	if this.RaftAdvertise == "" {
		this.RaftAdvertise = this.RaftBind
	}
	return nil

}

// ForceRead reads configuration from given file name or bails out if it fails
func ForceRead(fileName string) *Configuration {
	_, err := read(fileName)
	if err != nil {
		log.Fatal("Cannot read config file:", fileName, err)
	}
	readFileNames = []string{fileName}
	return Config
}

// Read reads configuration from zero, either, some or all given files, in order of input.
// A file can override configuration provided in previous file.
func Read(fileNames ...string) *Configuration {
	for _, fileName := range fileNames {
		read(fileName)
	}
	readFileNames = fileNames
	return Config
}

// read reads configuration from given file, or silently skips if the file does not exist.
// If the file does exist, then it is expected to be in valid JSON format or the function bails out.
func read(fileName string) (*Configuration, error) {
	if fileName == "" {
		return Config, fmt.Errorf("Empty file name")
	}
	file, err := os.Open(fileName)
	if err != nil {
		return Config, err
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(Config)
	if err == nil {
		log.Infof("Read config: %s", fileName)
	} else {
		log.Fatal("Cannot read config file:", fileName, err)
	}
	if err := Config.postReadAdjustments(); err != nil {
		log.Fatale(err)
	}
	return Config, err
}

// MarkConfigurationLoaded is called once configuration has first been loaded.
// Listeners on ConfigurationLoaded will get a notification
func MarkConfigurationLoaded() {
	go func() {
		for {
			configurationLoaded <- true
		}
	}()
	// wait for it
	<-configurationLoaded
}
