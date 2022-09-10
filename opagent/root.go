package opagent

import (
	"op-agent/config"
	"github.com/openark/golib/log"
	"github.com/spf13/cobra"
)

var (
	rootCmd    *cobra.Command
	configFile string
)

func init() {
	cobra.EnableCommandSorting = false
	rootCmd = &cobra.Command{
		Use:   "opagent",
		Short: "MySQL agent command line operation tool",
		Long: `Example:
			opagent get nodes [IP|HOSTNAME] [--o wide]
			opagent get jobs [JOBNAME]
			opagent get packages [PACKAGENAME]
			opagent logs <JOBNAME> <HOSTIP|ALL> [LIMIT] [--o wide|short]
			opagent upload <FILE|DOCUMENT> [deploymentDirName]
			opagent download <PACKAGENAME> [VERSION]
			opagent rollback <PACKAGENAME> <VERSION>
			opagent save <task.json>
			opagent top <JOBNAME>
			opagent analysis <packages|jobs> [jobName] [onceJobVersion]
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
		config.Read("./.opagent.conf.json" ,"/etc/opagent.conf.json", "conf/opagent.conf.json", "opagent.conf.json")
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
