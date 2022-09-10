package db

import (
	sql2 "database/sql"
)

func Insert(sql string) (int64,error) {
	var (
		db *sql2.DB
		err error
		insertId int64
		result sql2.Result
	)
	if db, err = OpenDb();err != nil {
		return  insertId,err
	}
	if result,err = db.Exec(sql); err != nil {
		return insertId,err
	}
	insertId,err = result.LastInsertId()
	return insertId, err
}

func ExecUpdate(sql string)  (int64,error){
	var (
		db *sql2.DB
		err error
		result sql2.Result
		rowsAffect int64

	)
	if db, err = OpenDb();err != nil {
		return  rowsAffect,err
	}
	if result,err = db.Exec(sql); err != nil {
		return rowsAffect,err
	}
	rowsAffect, err = result.RowsAffected()
	return rowsAffect, err
}