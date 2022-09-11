package agentCli

import (
	"op-agent/config"
	"github.com/openark/golib/log"
	"github.com/spf13/cobra"

	_ "github.com/go-sql-driver/mysql"
)

var (
	rootCmd    *cobra.Command
	configFile string
)

func init() {
	cobra.EnableCommandSorting = false
	rootCmd = &cobra.Command{
		Use:   "agentCli",
		Short: "MySQL agent command line operation tool",
		Long: `Example:
			agentCli get nodes [IP|HOSTNAME] [--o wide]
			agentCli get jobs [JOBNAME]
			agentCli get packages [PACKAGENAME]
			agentCli logs <JOBNAME> <HOSTIP|ALL> [LIMIT] [--o wide|short]
			agentCli upload <FILE|DOCUMENT> [deploymentDirName]
			agentCli download <PACKAGENAME> [VERSION]
			agentCli rollback <PACKAGENAME> <VERSION>
			agentCli save <task.json>
			agentCli top <JOBNAME>
			agentCli analysis <packages|jobs> [jobName] [onceJobVersion]
		`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       "1.0.0",
	}
	rootCmd.AddCommand(
		newGet(),
		newLogs(),
		newSave(),
		newLogs(),
		newDownload(),
		newUpload(),
		newRollback(),
		newAnalysis(),
	)
	rootCmd.PersistentFlags().StringVar((*string)(&configFile), "config", "", "config file name.")
	if len(configFile) > 0 {
		config.ForceRead(configFile)
	} else {
		config.Read("./.agentCli.conf.json" ,"/etc/agentCli.conf.json", "conf/agentCli.conf.json", "agentCli.conf.json")
	}
	config.MarkConfigurationLoaded()
}

// Execute executes the root command
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Errorf("%v", err)
	}
}
