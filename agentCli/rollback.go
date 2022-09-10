package agentCli

import (
        "fmt"
        "github.com/openark/golib/log"
        "github.com/spf13/cobra"
)


func newRollback() *cobra.Command {
        var (
        	packageName string
        	packageVersion string
        )

        cmd := &cobra.Command{
                Use:   "rollback <packageName> [VERSION]",
                Short: "rollback the specified version of the task package",
                Long:  `Example:
                        rollback <PackageName> [VERSION]
		`,
                SilenceUsage: true,
                RunE: func(cmd *cobra.Command, args []string) error {
                        if len(args) < 2 {
                                return fmt.Errorf("Package name and version must not be null ")
                        }
                        packageName = args[0]
			packageVersion = args[1]
                        if err := rollbackPackage(packageName, packageVersion); err != nil {
                                log.Errorf("%+v", err)
                        }
                        return nil
                },
        }
        return cmd
}

func rollbackPackage(packageName string, version string) error {
        return nil
}
