package agentCli

import (
	"errors"
	"fmt"
	"strings"

	"op-agent/db"
)

func ListOneJob(jobname string) (map[string]string, error) {
	var (
		query   string
		err     error
		results []map[string]string
	)
	query = fmt.Sprintf("select * from jobs where jobname = '%s'", jobname)
	if results, err = db.QueryAll(query); err != nil {
		errInfo := fmt.Sprintf("sql出错了,sql:%s,error:%s", query, err.Error())
		return nil, errors.New(errInfo)
	}
	if len(results) != 0 {
		return results[0], nil
	}
	return map[string]string{}, nil
}

func SaveJob(job map[string]string) error {
	var (
		columns         string
		values          string
		setColumns      string
		whereConditon   string
		err             error
		recordExistFlag bool
		count           int64
	)
	columns = "("
	values = "("
	if value, ok := job["jobname"]; ok {
		query := fmt.Sprintf("select count(1) from jobs where jobname = '%s'", value)
		if count, err = db.QueryCount(query); err != nil {
			return err
		}
		if count != 0 {
			recordExistFlag = true
		}
		columns = columns + "jobname" + ","
		values = values + fmt.Sprintf("'%s',", value)
		whereConditon = fmt.Sprintf("jobname='%s'", value)
	} else {
		err = fmt.Errorf("jobname can't be null")
		return err
	}
	if value, ok := job["command"]; ok {
		columns = columns + "command" + ","
		setColumns = setColumns + fmt.Sprintf("command='%s',", value)
		values = values + fmt.Sprintf("'%s',", value)
	}
	if value, ok := job["cronexpr"]; ok {
		columns = columns + "cronexpr" + ","
		setColumns = setColumns + fmt.Sprintf("cronexpr='%s',", value)
		values = values + fmt.Sprintf("'%s',", value)
	}
	if value, ok := job["timeout"]; ok {
		columns = columns + "timeout" + ","
		setColumns = setColumns + fmt.Sprintf("timeout='%s',", value)
		values = values + fmt.Sprintf("'%s',", value)
	}
	if value, ok := job["parameters"]; ok {
		columns = columns + "parameters" + ","
		setColumns = setColumns + fmt.Sprintf("parameters='%s',", value)
		values = values + fmt.Sprintf("'%s',", value)
	}
	if value, ok := job["outputflag"]; ok {
		columns = columns + "outputflag" + ","
		setColumns = setColumns + fmt.Sprintf("outputflag='%s',", value)
		values = values + fmt.Sprintf("'%s',", value)
	}
	if value, ok := job["whiteips"]; ok {
		columns = columns + "whiteips" + ","
		setColumns = setColumns + fmt.Sprintf("whiteips='%s',", value)
		values = values + fmt.Sprintf("'%s',", value)
	}
	if value, ok := job["blackips"]; ok {
		columns = columns + "blackips" + ","
		setColumns = setColumns + fmt.Sprintf("blackips='%s',", value)
		values = values + fmt.Sprintf("'%s',", value)
	}
	if value, ok := job["enabled"]; ok {
		columns = columns + "enabled" + ","
		setColumns = setColumns + fmt.Sprintf("enabled='%s',", value)
		values = values + fmt.Sprintf("'%s',", value)
	}
	columns = strings.TrimRight(columns, ",") + ")"
	setColumns = strings.TrimRight(setColumns, ",")
	values = strings.TrimRight(values, ",") + ")"
	updateSql := fmt.Sprintf("update jobs set %s where %s", setColumns, whereConditon)
	insertSql := fmt.Sprintf("insert into jobs%s values%s", columns, values)
	if recordExistFlag {
		if _, err := db.ExecUpdate(updateSql); err != nil {
			errInfo := fmt.Sprintf("sql出错了,sql:%s,error:%s", updateSql, err.Error())
			return errors.New(errInfo)
		}
	} else {
		if _, err = db.Insert(insertSql); err != nil {
			errInfo := fmt.Sprintf("sql出错了,sql:%s,error:%s", insertSql, err.Error())
			return errors.New(errInfo)
		}
	}
	return nil
}
