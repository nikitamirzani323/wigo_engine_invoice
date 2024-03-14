package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/nikitamirzani323/wigo_engine_invoice/configs"
	"github.com/nikitamirzani323/wigo_engine_invoice/db"
	"github.com/nikitamirzani323/wigo_engine_invoice/helpers"
)

func Get_counter(field_column string) int {
	con := db.CreateCon()
	ctx := context.Background()
	idrecord_counter := 0
	sqlcounter := `SELECT 
					counter 
					FROM ` + configs.DB_tbl_counter + ` 
					WHERE nmcounter = $1 
				`
	var counter int = 0
	row := con.QueryRowContext(ctx, sqlcounter, field_column)
	switch e := row.Scan(&counter); e {
	case sql.ErrNoRows:
		fmt.Println("COUNTER - No rows were returned!")
	case nil:
		fmt.Println(counter)
	default:
		panic(e)
	}
	if counter > 0 {
		idrecord_counter = int(counter) + 1
		stmt, e := con.PrepareContext(ctx, "UPDATE "+configs.DB_tbl_counter+" SET counter=$1 WHERE nmcounter=$2 ")
		helpers.ErrorCheck(e)
		res, e := stmt.ExecContext(ctx, idrecord_counter, field_column)
		helpers.ErrorCheck(e)
		a, e := res.RowsAffected()
		helpers.ErrorCheck(e)
		if a > 0 {
			fmt.Println("COUNTER - UPDATE")
		}
		defer stmt.Close()
	} else {
		stmt, e := con.PrepareContext(ctx, "insert into "+configs.DB_tbl_counter+" (nmcounter, counter) values ($1, $2)")
		helpers.ErrorCheck(e)
		res, e := stmt.ExecContext(ctx, field_column, 1)
		helpers.ErrorCheck(e)
		id, e := res.RowsAffected()
		helpers.ErrorCheck(e)
		fmt.Println("Insert id", id)
		fmt.Println("NEW")
		idrecord_counter = 1
		defer stmt.Close()
	}
	return idrecord_counter
}
func Get_listitemsearch(data, pemisah, search string) bool {
	flag := false
	temp := strings.Split(data, pemisah)
	for i := 0; i < len(temp); i++ {
		if temp[i] == search {
			flag = true
			break
		}
	}
	return flag
}
func CheckDB(table, field, value string) bool {
	con := db.CreateCon()
	ctx := context.Background()
	flag := false
	sql_db := `SELECT 
					` + field + ` 
					FROM ` + table + ` 
					WHERE ` + field + ` = $1 
				`
	row := con.QueryRowContext(ctx, sql_db, value)
	switch e := row.Scan(&field); e {
	case sql.ErrNoRows:
		fmt.Println("CHECK DB - No rows were returned!")
		flag = false
	case nil:
		flag = true
	default:
		flag = false
	}
	return flag
}
func CheckDBTwoField(table, field_1, value_1, field_2, value_2 string) bool {
	con := db.CreateCon()
	ctx := context.Background()
	flag := false
	sql_db := `SELECT 
					` + field_1 + ` 
					FROM ` + table + ` 
					WHERE ` + field_1 + ` = $1 
					AND ` + field_2 + ` = $2
				`
	row := con.QueryRowContext(ctx, sql_db, value_1, value_2)
	switch e := row.Scan(&field_1); e {
	case sql.ErrNoRows:
		fmt.Println("CHECKDBTWOFIELD - No rows were returned!")
		flag = false
	case nil:
		flag = true
	default:
		flag = false
	}
	return flag
}
func CheckDBThreeField(table, field_1, value_1, field_2, value_2, field_3, value_3 string) bool {
	con := db.CreateCon()
	ctx := context.Background()
	flag := false
	sql_db := `SELECT 
					` + field_1 + ` 
					FROM ` + table + ` 
					WHERE ` + field_1 + ` = $1 
					AND ` + field_2 + ` = $2 
					AND ` + field_3 + ` = $3 
				`
	row := con.QueryRowContext(ctx, sql_db, value_1, value_2, value_3)
	switch e := row.Scan(&field_1); e {
	case sql.ErrNoRows:
		fmt.Println("CHECKDBTHREEFIELD - No rows were returned!")
		flag = false
	case nil:
		flag = true
	default:
		flag = false
	}
	return flag
}
func Get_AdminRule(tipe, idadmin string) string {
	con := db.CreateCon()
	ctx := context.Background()
	flag := false
	result := ""
	ruleadmingroup := ""

	sql_select := `SELECT
		ruleadmingroup  
		FROM ` + configs.DB_tbl_admingroup + `  
		WHERE idadmin = $1 
	`
	row := con.QueryRowContext(ctx, sql_select, idadmin)
	switch e := row.Scan(&ruleadmingroup); e {
	case sql.ErrNoRows:
		flag = false
	case nil:
		flag = true

	default:
		panic(e)
	}
	if flag {
		switch tipe {
		case "ruleadmingroup":
			result = ruleadmingroup
		}
	}
	return result
}
func Delete_SQL(sql, table string, args ...interface{}) bool {
	con := db.CreateCon()
	ctx := context.Background()
	flag := false
	stmt_delete, e_delete := con.PrepareContext(ctx, sql)
	helpers.ErrorCheck(e_delete)
	defer stmt_delete.Close()
	rec_delete, e_delete := stmt_delete.ExecContext(ctx, args...)

	helpers.ErrorCheck(e_delete)
	deletesource, e := rec_delete.RowsAffected()
	helpers.ErrorCheck(e)
	if deletesource > 0 {
		flag = true
		fmt.Printf("Data %s Berhasil di delete", table)
	} else {
		fmt.Printf("Data %s Failed di delete", table)
	}
	return flag
}
func Exec_SQL(sql, table, action string, args ...interface{}) (bool, string) {
	con := db.CreateCon()
	ctx := context.Background()
	flag := false
	msg := ""

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	stmt_exec, e_exec := con.PrepareContext(ctx, sql)
	helpers.ErrorCheck(e_exec)
	defer stmt_exec.Close()
	rec_exec, e_exec := stmt_exec.ExecContext(ctx, args...)

	helpers.ErrorCheck(e_exec)
	exec, e := rec_exec.RowsAffected()
	helpers.ErrorCheck(e)
	if exec > 0 {
		flag = true
		msg = "Data " + table + " Berhasil di " + action
	} else {
		msg = "Data " + table + " Failed di " + action
	}
	return flag, msg
}
func Get_mappingdatabase(codeagen string) (string, string, string, string) {
	tbl_mst_config := `"db_wigo_` + strings.ToLower(codeagen) + `".tbl_mst_config`
	tbl_trx_transaksi := `"db_wigo_` + strings.ToLower(codeagen) + `".tbl_trx_transaksi_12digit`
	tbl_trx_transaksidetail := `"db_wigo_` + strings.ToLower(codeagen) + `".tbl_trx_transaksidetail_12digit`
	view_invoice_client := `"db_wigo_` + strings.ToLower(codeagen) + `".view_invoice_client`

	return tbl_mst_config, tbl_trx_transaksi, tbl_trx_transaksidetail, view_invoice_client
}
