
package opagent

import (
	"fmt"
	"op-agent/db"
	"github.com/openark/golib/log"
	"github.com/openark/golib/sqlutils"
	"github.com/spf13/cobra"
)


func newPackages() *cobra.Command {
	var packageName string
	cmd := &cobra.Command{
		Use: "packages",
		Short: "View packages information",
		Long: `Example:
			packages [package name]
		`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				packageName = args[0]
			}
			if err := GetPackagesInfo(packageName); err != nil {
				log.Errorf("GetPackagesInfo('%s') err: %s+v", packageName, err)
			}
			return nil
		},
	}
	return cmd
}


func GetPackagesInfo(packageName string) error {
	var (
		err error
		query string
	)
	if packageName != "" {
		query = `select  
					package_name, package_version,  _timestamp, deploydir
				from package_info where package_name='%s' order by package_version`
		query = fmt.Sprintf(query, packageName)
	} else {
		query = `select 
					package_name, max(package_version) as package_version, _timestamp, deploydir 
				from  package_info group by package_name`
	}
	dataLists := [][]string{}
	err = db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
		resultList := []string{}
		resultList = append(resultList, m.GetString("package_name"))
		resultList = append(resultList, m.GetString("package_version"))
		resultList = append(resultList, m.GetString("_timestamp"))
		resultList = append(resultList, m.GetString("deploydir"))
		dataLists = append(dataLists, resultList)
		return nil
	})
	if err != nil {
		return err
	}
	table := TableWriter([]string{"packageName", "packageVersion", "timestamp", "deploydir"}, dataLists)
	table.Render()
	return nil
}
