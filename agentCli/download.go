package agentCli

import (
        "fmt"
        "io/ioutil"
        "strings"

        "github.com/openark/golib/log"
        "github.com/spf13/cobra"

        "op-agent/db"
        "op-agent/util"
)


func newDownload() *cobra.Command {
        var (
        	packageName string
        	packageVersion string
        )

        cmd := &cobra.Command{
                Use:   "download <packageName> [VERSION]",
                Short: "Download the specified version of the task package",
                Long:  `Example:
                        download <packageName> [VERSION]
		`,
                SilenceUsage: true,
                RunE: func(cmd *cobra.Command, args []string) error {
                        if len(args) == 0 {
                                return fmt.Errorf("Must specify package name ")
                        }
                        packageName = args[0]
                        if len(args) > 1 {
                                packageVersion = args[1]
                        }
                        if err := downloadFile(packageName, packageVersion); err != nil {
                                log.Errorf("%+v", err)
                        }
                        return nil
                },
        }
        return cmd
}


func downloadFile(packageName string, version string) error {
        var (
                query string
                packageContent []byte
        )

        if version == "" {
                query = fmt.Sprintf("select package_content from package_info where  package_name = '%s'  order by package_version desc limit 1 ", packageName)
        } else {
                query = fmt.Sprintf("select package_content from package_info where  package_name = '%s' and version = '%s' limit 1", packageName, version)
        }

        rows, err := db.DBQueryAll(query)
        if err != nil {
                return fmt.Errorf("%s %+v", query, err)
        }

        for rows.Next() {
                err := rows.Scan(&packageContent)
                if err != nil {
                        return  log.Errore(err)
                }
        }

        if len(packageContent) == 0 {
                return log.Errorf("%s result is null", query)
        }

        whereToPlacePackage := "./" + packageName
        err = ioutil.WriteFile(whereToPlacePackage, packageContent, 0644)
        if err != nil {
                return fmt.Errorf("ioutil.WriteFile(%s) : %+v", whereToPlacePackage, err)
        }
        if strings.HasSuffix(packageName, ".tar.gz") {
                cmd := "tar xvf " + whereToPlacePackage
                // tar xvf project.tar.gz
                if err := util.RunCommandNoOutput(cmd); err != nil {
                        return fmt.Errorf("RunCommand %s : %s+v", cmd, err)
                }
        }
        log.Infof("Download file %s succeeded!!!", packageName)
        return nil
}
