package agentCli

import (
	"github.com/openark/golib/log"
	"github.com/spf13/cobra"
)

var wide string

func newNodes() *cobra.Command {
	var (
		host string
	)
	cmd := &cobra.Command{
		Use: "nodes [IP|HOSTNAME] [--o wide]",
		Short: "View nodes information",
		Long: `Example:
			nodes [IP|HOSTNAME] [--o wide]
		`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				host = args[0]
			}
			if err := GetNodesStatus(host, wide); err != nil {
				log.Errorf("GetNodesStatus('%s') err: %s+v", host, err)
			}
			return nil
		},
	}
	cmd.PersistentFlags().StringVar((*string)(&wide), "o", "", "Show details.")
	return cmd
}

func GetNodesStatus(host string, wide string) error {
	dataLists := TakeAgentsStatus(host, wide)
	title := make([]string, 0)
	if host != "" || wide == "wide" {
		title = []string{"HostName", "IP", "Status", "Version", "FirstSeen", "LastSeen", "Packages-On-This-Host", "All-Packages"}
	} else {
		title = []string{"HostName", "IP", "Status", "Version", "FirstSeen", "LastSeen"}
	}
	table := TableWriter(title, dataLists)
	table.Render()
	return nil
}