package agentCli

import (
	"github.com/openark/golib/log"
	"github.com/spf13/cobra"
	"strings"
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
	activeAgentLists := make([][]string, 0)
	nonActiveAgentLists := make([][]string, 0)
	dataMapLists := TakeAgentsStatus(host, wide)
	for _, data := range dataMapLists {
		activeLists := make([]string,0)
		nonActiveLists := make([]string,0)
		if data["status"] == "Ready" {
			activeLists = append(activeLists, data["hostname"])
			activeLists = append(activeLists, strings.Replace(data["ip"],"," ,"\n", -1 ))
			activeLists = append(activeLists, data["status"])
			activeLists = append(activeLists, data["app_version"])
			activeLists = append(activeLists, data["first_seen_active"])
			activeLists = append(activeLists, data["last_seen_active"])
			if wide == "wide" {
				activeLists = append(activeLists, data["hostPackages"])
				activeLists = append(activeLists, data["allNewPackages"])
			}
			activeAgentLists = append(activeAgentLists, activeLists)
		}
		if data["status"] == "NotReady" {
			nonActiveLists = append(nonActiveLists, data["hostname"])
			nonActiveLists = append(nonActiveLists, strings.Replace(data["ip"],"," ,"\n", -1 ))
			nonActiveLists = append(nonActiveLists, data["status"])
			nonActiveLists = append(nonActiveLists, data["app_version"])
			nonActiveLists = append(nonActiveLists, data["first_seen_active"])
			nonActiveLists = append(nonActiveLists, data["last_seen_active"])
			if wide == "wide" {
				nonActiveLists = append(nonActiveLists, data["hostPackages"])
				nonActiveLists = append(nonActiveLists, data["allNewPackages"])
			}
			nonActiveAgentLists = append(nonActiveAgentLists, nonActiveLists)
		}
	}

	title := make([]string, 0)
	if host != "" || wide == "wide" {
		title = []string{"HostName", "IP", "Status", "Version", "FirstSeen", "LastSeen", "Packages-On-This-Host", "All-Packages"}
	} else {
		title = []string{"HostName", "IP", "Status", "Version", "FirstSeen", "LastSeen"}
	}
	table := TableWriter(title, append(activeAgentLists, nonActiveAgentLists...))
	table.Render()
	return nil
}