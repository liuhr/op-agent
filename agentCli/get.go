package agentCli

import (
        "github.com/spf13/cobra"
)


func newGet() *cobra.Command {
        cmd := &cobra.Command{
                Use: "get <nodes|jobs|packages>",
                Short: "View <nodes|jobs|packages> information",
                Long: `Example:
                    get nodes [IP|HOSTNAME] [--o wide]
                    get jobs [JOBNAME] 
                    get packages [PACKAGENAME]
		`,
                SilenceUsage: true,
        }
        cmd.AddCommand(
              newNodes(),
              newJobs(),
              newPackages(),
        )
        return cmd
}
