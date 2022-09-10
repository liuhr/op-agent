package db

import (
	"database/sql"
	"github.com/openark/golib/log"
)

func QueryAll(query string) ([]map[string]string,error) {
	var (
		db *sql.DB
		err error
		rows *sql.Rows
		result []map[string]string
	)
	result = make([]map[string]string, 0)
	if db, err = OpenDb();err != nil {
		return  result,err
	}
	rows, err = db.Query(query)
	if nil != err {
		log.Info("db.Query err:%s", err,query)
		return result, err
	}
	result,err = queryAllRows(rows)
	return result,err
}

func queryAllRows(rows *sql.Rows) ([]map[string]string,error) {
	result := make([]map[string]string,0)
	defer func(rows *sql.Rows) {
		if rows != nil {
			rows.Close()
		}
	}(rows)

	columnsName, err := rows.Columns()
	if nil != err {
		log.Info("rows.Columns err:", err)
		return result,err
	}

	values := make([]sql.RawBytes, len(columnsName))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if nil != err {
			log.Info("rows.Next err:", err)
		}

		row_map := make(map[string]string)
		for i, col := range values {
			if col == nil {
				row_map[columnsName[i]] = "NULL"
			} else {
				row_map[columnsName[i]] = string(col)
			}
		}
		result = append(result, row_map)
	}

	err = rows.Err()
	if nil != err {
		log.Info("rows.Err:", err)
	}
	return result,nil
}


func QueryCount(query string) (int64,error) {
	var (
		count int64
		db *sql.DB
		err error
		rows *sql.Row
	)
	if db, err = OpenDb();err != nil {
		return  count,err
	}
	rows = db.QueryRow(query)
	err = rows.Scan(&count)
	return count, err
}





