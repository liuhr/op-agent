package agent

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/openark/golib/log"

	"op-agent/config"
	"op-agent/db"
	"op-agent/process"
	"op-agent/util"
)

type PluginManager struct {}


type FileContent struct {
	deploydir		string
	md5sum          string
	pluginContent []byte
}

var pluginManager *PluginManager


func (plugins *PluginManager) ContinuesWatchPluginsChange(pluginTask map[string]string) error {
	var (
		err error
	)
	if pluginTask == nil {
		return nil
	}
	if len(pluginTask) == 0 {
		return nil
	}
	err = plugins.downloadFile(pluginTask)
	if err != nil {
		update := `
			update agent_package_task set status = '4', fail_reason=? 
			where token=? and package_name=? and package_version=?
		`
		if _, err := db.ExecDb(update, err.Error(), process.ThisHostToken, pluginTask["packageName"], pluginTask["packageVersion"]); err != nil {
			log.Errorf("%s %+v", update, err)
		}
		return err
	}

	update := `
		update agent_package_task set status = '3'
		where token=? and package_name=? and package_version=?
	`
	if _, err := db.ExecDb(update, process.ThisHostToken, pluginTask["packageName"], pluginTask["packageVersion"]); err != nil {
		log.Errorf("%s %+v", update, err)
		return err
	}
	return nil
}

func (plugins *PluginManager) downloadFile(pluginMap map[string]string) error {
	var (
		results []FileContent
	)
	results = make([]FileContent,0)


	query := `
			select 
				md5sum,package_content,deploydir
			from package_info where package_name='%s' and package_version='%s'
	`
	query = fmt.Sprintf(query, pluginMap["packageName"],pluginMap["packageVersion"])
	rows, err := db.DBQueryAll(query)
	if err != nil {
		return fmt.Errorf("%s %+v", query, err)
	}
	for rows.Next() {
		var file FileContent
		err := rows.Scan(&file.md5sum, &file.pluginContent, &file.deploydir)
		if err != nil {
			return  log.Errore(err)
		}
		results = append(results, file)
	}
	if len(results) == 0 {
		return log.Errorf("%s result is null", query)
	}
	data := results[0].pluginContent
	deploymentDir := results[0].deploydir

	if deploymentDir == "" {
		deploymentDir = config.Config.PluginDeploymentDir // ./src
	}
	if _, err = util.MakeDir(deploymentDir); err != nil {
		log.Errorf("MakeDir %s: %+v", deploymentDir, err)
	}

	whereToPlacePackage := deploymentDir + "/" + pluginMap["packageName"]  // ./src/project.tar.gz
	err = ioutil.WriteFile(whereToPlacePackage, data, 0644)
	if err != nil {
		return fmt.Errorf("ioutil.WriteFile(%s) : %+v", whereToPlacePackage, err)
	}
	log.Infof("Write file succeeded: %s", whereToPlacePackage)
	cmd := ""
	if strings.HasSuffix(pluginMap["packageName"], ".tar.gz") {
		cmd = "tar xvf " + whereToPlacePackage + " -C" + " " + deploymentDir + "/"
		// tar xvf ./src/project.tar.gz -C ./src/
	} else {
		cmd = "chmod +x " +  whereToPlacePackage

	}
	if cmd != "" {
		if err := util.RunCommandNoOutput(cmd); err != nil {
			return fmt.Errorf("RunCommand %s : %s+v", cmd, err)
		}
	}
	return nil
}

func WatchPluginVersion(packageTask map[string]string) error {
	if err := pluginManager.ContinuesWatchPluginsChange(packageTask); err != nil {
		log.Errorf("pluginManager.ContinuesWatchPluginsChange err %+v", err)
		return err
	}
	return nil
}
