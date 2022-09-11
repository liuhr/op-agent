package agentCli

import (
	"database/sql"
	"fmt"
	"github.com/openark/golib/log"
	"github.com/openark/golib/sqlutils"
	"github.com/spf13/cobra"
	"op-agent/db"
	"op-agent/util"
	"path/filepath"
	"time"
)


func newUpload() *cobra.Command {
	var (
		err error
		uploadFile string
		deploymentDir string
		result map[string]string
	)
	cmd := &cobra.Command{
		Use:	"upload <file|document> [deploymentDirName]",
		Short:  "Upload task package or executable file",
		Long:   `Example:
			upload test.py|projectDir /data/my-agent/src/
		`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("upload file or project dir can not be null")
			}
			uploadFile = args[0]
			if len(args) > 1 {
				deploymentDir = args[1]
			}
			if !util.PathExists(uploadFile) {
				return fmt.Errorf("%s :The specified file or directory does not exist", uploadFile)
			}
			if result, err = uploadFunc(uploadFile, deploymentDir); err != nil {
				return err
			}
			log.Infof("File uploaded successfully！！！！！")
			if len(result) != 0 {
				log.Infof("package_name: %s, package_version: %s, md5sum: %s, package_desc: %s, _timestamp: %s",
					result["package_name"],result["package_version"],result["md5sum"],result["package_desc"],result["_timestamp"])
			}
			return nil
			},
	}
	return cmd
}


func uploadFunc(file string, deploymentDir string) (map[string]string, error) {
	var (
		//count int64
		rows int64
		query string
		file_md5 string
		package_name string
		package_content []byte
		package_size	int64
		sqlResult sql.Result
		result map[string]string
	)
	result = make(map[string]string,0)
	isdir, err := util.IsDir(file)
	if err != nil {
		return result, fmt.Errorf("util.IsDir(%s) err: %+v", file,err)
	}
	base := filepath.Base(file)
	if isdir {
		zip_filename := base + ".tar.gz"
		package_name = zip_filename
		if err := util.Zip(file, zip_filename); err != nil {
			return result, fmt.Errorf("util.Zip(%s, %s) err: %+v", file, zip_filename, err)
		}
		if file_md5, err = util.GetFileMd5(zip_filename); err != nil {
			return result, fmt.Errorf("util.GetFileMd5(%s) err: %+v", zip_filename, err)
		}
		if package_content, err = util.ReadFile(zip_filename); err != nil {
			return result, fmt.Errorf("util.ReadFile(%s) err: %+v", zip_filename, err)
		}
		package_size, _ = util.GetFileSize(zip_filename)
	} else {
		package_name = base
		if file_md5, err = util.GetFileMd5(file); err != nil {
			return result, fmt.Errorf("util.GetFileMd5(%s) err: %+v", file, err)
		}
		package_content, _ = util.ReadFile(file)
		package_size, _ = util.GetFileSize(file)
	}
	timeUnix := time.Now().Unix()
	/*
	query := fmt.Sprintf("select id from package_info where package_name='%s' and md5sum='%s'", package_name, file_md5)
	count, _ = db.QueryCount(query)
	if count > 0 {
		return result, fmt.Errorf("Package %s exists and content didn't change! ", package_name)
	}*/
	if deploymentDir == "" {
		query = fmt.Sprintf("select deploydir from  package_info where package_name='%s' order by package_version desc limit 1")
		db.QueryDBRowsMap(query, func(m sqlutils.RowMap) error {
			deploymentDir = m.GetString("deploydir")
			return nil
		})
	}

	insert := `
			insert ignore into package_info
				(package_name, deployDir, md5sum,package_version,package_content,package_size,ctime)
			values
				(?, ?, ?, ?, ?, ?, now())
	`
	sqlResult, err = db.ExecDb(insert, package_name,  deploymentDir, file_md5, timeUnix, package_content, package_size,)
	if err != nil {
		return result, fmt.Errorf("%+v", err)
	}
	if rows, err = sqlResult.RowsAffected(); err != nil {
		return result, fmt.Errorf("%+v", err)
	}
	if rows == 0 {
		return  result, fmt.Errorf("upload package %s is not effective", package_name)
	}

	query = fmt.Sprintf(`
								select 
									package_name,md5sum,package_version,package_desc,_timestamp 
								from 
									package_info 
								where 
									package_name='%s' and md5sum='%s'
								`, package_name, file_md5)
	resultsMap, _ := db.QueryAll(query)
	if len(resultsMap) > 0 {
		result = resultsMap[0]
	}
	return result, nil
}

