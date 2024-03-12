package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/go-redis/redis"
	"github.com/joho/godotenv"
	"github.com/nikitamirzani323/wigo_engine_invoice/db"
	"github.com/nikitamirzani323/wigo_engine_invoice/helpers"
	"github.com/nikitamirzani323/wigo_engine_invoice/models"
	"github.com/nleeper/goment"
)

const invoice_client_redis = "CLIENT_LISTINVOICE"
const invoice_result_redis = "CLIENT_RESULT"
const invoice_agen_redis = "LISTINVOICE_2D30S_AGEN"

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Failed to load env file")
	}
	db.Init()

	dbHost := os.Getenv("DB_REDIS_HOST") + ":" + os.Getenv("DB_REDIS_PORT")
	dbPass := os.Getenv("DB_REDIS_PASSWORD")
	dbName, _ := strconv.Atoi(os.Getenv("DB_REDIS_NAME"))

	rdb := redis.NewClient(&redis.Options{
		Addr:     dbHost,
		Password: dbPass,
		DB:       dbName,
	})

	// resultredis := rdb.Subscribe("", "payload_nuke")
	resultredis := rdb.Subscribe("", "payload_enginesave_nuke")

	for {
		msg, err := resultredis.ReceiveMessage()
		if err != nil {
			panic(err)
		}

		// fmt.Println("Received message from " + msg.Payload + " channel.")
		// data_send = invoice + "|" + prize_2D + "|" + idcompany
		//invoce|result|company
		msg_sse := msg.Payload

		msg_replace := strings.Replace(msg_sse, `"`, "", -1)
		msg_split := strings.Split(msg_replace, "|")

		invoice := msg_split[0]
		result := msg_split[1]
		company := msg_split[2]
		// fmt.Printf("%s-%s-%s\n", invoice, result, company)
		fmt.Printf("%s\n", msg_replace)

		if company != "" && invoice != "" {
			if result != "" {
				Update_transaksi(company, invoice, result)
			}
		}

		// time.Sleep(1 * time.Second)

	}

	c := make(chan os.Signal, 1)                    // Create channel to signify a signal being sent
	signal.Notify(c, os.Interrupt, syscall.SIGTERM) // When an interrupt or termination signal is sent, notify the channel

	_ = <-c // This blocks the main thread until an interrupt is received
	fmt.Println("Gracefully shutting down...")

	fmt.Println("Running cleanup tasks...")

	// Your cleanup tasks go here
	// db.Close()
	// redisConn.Close()
	rdb.Close()
	fmt.Println("Fiber was successful shutdown.")
}
func Update_transaksi(idcompany, invoice, result string) {
	msg := "Failed"
	tglnow, _ := goment.New()
	// id_invoice := _GetInvoice(idcompany)
	// prize_2D := helpers.GenerateNumber(2)
	// flag_compile := false

	_, tbl_trx_transaksi, tbl_trx_transaksidetail, _ := models.Get_mappingdatabase(idcompany)

	con := db.CreateCon()
	ctx := context.Background()
	flag_detail := false
	sql_select_detail := `SELECT 
					idtransaksidetail , nomor, tipebet,bet, multiplier, username_client 
					FROM ` + tbl_trx_transaksidetail + `  
					WHERE status_transaksidetail='RUNNING'  
					AND idtransaksi='` + invoice + `'  `

	row, err := con.QueryContext(ctx, sql_select_detail)
	helpers.ErrorCheck(err)
	for row.Next() {
		var (
			bet_db                                                         int
			multiplier_db                                                  float64
			idtransaksidetail_db, nomor_db, tipebet_db, username_client_db string
		)

		err = row.Scan(&idtransaksidetail_db, &nomor_db, &tipebet_db, &bet_db, &multiplier_db, &username_client_db)
		helpers.ErrorCheck(err)

		status_client := _rumuswigo(tipebet_db, nomor_db, result)
		win := 0
		if status_client == "WIN" {
			win = bet_db + int(float64(bet_db)*multiplier_db)
		}

		// UPDATE STATUS DETAIL
		sql_update_detail := `
					UPDATE 
					` + tbl_trx_transaksidetail + `  
					SET status_transaksidetail=$1, win=$2, 
					update_transaksidetail=$3, updatedate_transaksidetail=$4           
					WHERE idtransaksidetail=$5          
				`
		flag_update_detail, msg_update_detail := models.Exec_SQL(sql_update_detail, tbl_trx_transaksidetail, "UPDATE",
			status_client, win,
			"SYSTEM", tglnow.Format("YYYY-MM-DD HH:mm:ss"), idtransaksidetail_db)

		if !flag_update_detail {
			fmt.Println(msg_update_detail)
		}
		flag_detail = true

		key_redis_invoice_client := invoice_client_redis + "_" + strings.ToLower(idcompany) + "_" + strings.ToLower(username_client_db)
		val_invoice_client := helpers.DeleteRedis(key_redis_invoice_client)
		fmt.Println("")
		fmt.Printf("Redis Delete INVOICE : %d - %s \r", val_invoice_client, key_redis_invoice_client)
		fmt.Println("")
	}
	defer row.Close()
	if flag_detail {
		// UPDATE PARENT
		total_member := _GetTotalMember_Transaksi(tbl_trx_transaksidetail, invoice)
		total_bet, total_win := _GetTotalBetWin_Transaksi(tbl_trx_transaksidetail, invoice)
		sql_update_parent := `
					UPDATE 
					` + tbl_trx_transaksi + `  
					SET total_bet=$1, total_win=$2, total_member=$3,
					update_transaksi=$4, updatedate_transaksi=$5            
					WHERE idtransaksi=$6        
				`
		flag_update_parent, msg_update_parent := models.Exec_SQL(sql_update_parent, tbl_trx_transaksi, "UPDATE",
			total_bet, total_win, total_member,
			"SYSTEM", tglnow.Format("YYYY-MM-DD HH:mm:ss"), invoice)

		if !flag_update_parent {
			fmt.Println(msg_update_parent)

		} else {
			// flag_compile = true
			msg = "Success - Update Paret - " + invoice
		}
	}

	key_redis_result := invoice_result_redis + "_" + strings.ToLower(idcompany)
	val_result := helpers.DeleteRedis(key_redis_result)
	fmt.Println("")
	fmt.Printf("Redis Delete RESULT : %d - %s \n", val_result, key_redis_result)
	fmt.Println("")
	for i := 0; i <= 1000; i = i + 250 {
		//LISTINVOICE_2D30S_AGEN_nuke_0_
		key_redis_ageninvoice := invoice_agen_redis + "_" + strings.ToLower(idcompany) + "_" + strconv.Itoa(i) + "_"
		val_result := helpers.DeleteRedis(key_redis_ageninvoice)
		fmt.Printf("Redis Delete AGEN INVOICE : %d - %s \n", val_result, key_redis_ageninvoice)
	}

	// key_redis_detail := "LISTINVOICE_2D30S_AGEN_nuke_DETAIL_240312231346_WIN"
	key_redis_detail_win := invoice_agen_redis + "_" + strings.ToLower(idcompany) + "_DETAIL_" + invoice + "_WIN"
	key_redis_detail_lose := invoice_agen_redis + "_" + strings.ToLower(idcompany) + "_DETAIL_" + invoice + "_LOSE"
	key_redis_detail_running := invoice_agen_redis + "_" + strings.ToLower(idcompany) + "_DETAIL_" + invoice + "_RUNNING"
	val_detail_win := helpers.DeleteRedis(key_redis_detail_win)
	val_detail_lose := helpers.DeleteRedis(key_redis_detail_lose)
	val_detail_running := helpers.DeleteRedis(key_redis_detail_running)
	fmt.Println("")
	fmt.Printf("Redis Delete DETAIL WIN : %d\n", val_detail_win)
	fmt.Printf("Redis Delete DETAIL LOSE : %d\n", val_detail_lose)
	fmt.Printf("Redis Delete DETAIL RUNNIN : %d\n", val_detail_running)
	fmt.Println("")
	// return flag_compile
	fmt.Println(msg)
}

func _GetTotalBetWin_Transaksi(table, idtransaksi string) (int, int) {
	con := db.CreateCon()
	ctx := context.Background()
	total_bet := 0
	total_win := 0
	sql_select := ""
	sql_select += "SELECT "
	sql_select += "SUM(bet) AS total_bet, SUM(win) AS total_win  "
	sql_select += "FROM " + table + " "
	sql_select += "WHERE idtransaksi='" + idtransaksi + "'   "
	sql_select += "AND status_transaksidetail !='RUNNING'   "

	row := con.QueryRowContext(ctx, sql_select)
	switch e := row.Scan(&total_bet, &total_win); e {
	case sql.ErrNoRows:
	case nil:
	default:
		helpers.ErrorCheck(e)
	}

	return total_bet, total_win
}
func _GetTotalMember_Transaksi(table, idtransaksi string) int {
	con := db.CreateCon()
	ctx := context.Background()
	total_member := 0
	sql_select := ""
	sql_select += "SELECT "
	sql_select += "COUNT(distinct(username_client))  AS total_member  "
	sql_select += "FROM " + table + " "
	sql_select += "WHERE idtransaksi='" + idtransaksi + "'   "

	row := con.QueryRowContext(ctx, sql_select)
	switch e := row.Scan(&total_member); e {
	case sql.ErrNoRows:
	case nil:
	default:
		helpers.ErrorCheck(e)
	}

	return total_member
}
func _rumuswigo(tipebet, nomorclient, nomorkeluaran string) string {
	result := "LOSE"

	switch tipebet {
	case "ANGKA":
		if nomorclient == nomorkeluaran {
			result = "WIN"
		}
	case "REDBLACK":
		keluaran_ganjilgenap := _genapganjil(nomorkeluaran)
		keluaran_besarkecil := _besarkecil(nomorkeluaran)

		if nomorclient == keluaran_ganjilgenap {
			result = "WIN"
		}
		if nomorclient == keluaran_besarkecil {
			result = "WIN"
		}
	case "LINE":
		keluaran_line := _line(nomorkeluaran)
		if nomorclient == keluaran_line {
			result = "WIN"
		}
	}

	return result
}
func _genapganjil(nomorkeluaran string) string {
	nomor_generator := ""
	result := ""
	for i := 0; i <= 99; i++ {
		if i < 10 {
			nomor_generator = "0" + strconv.Itoa(i)
		} else {
			nomor_generator = strconv.Itoa(i)
		}
		if i%2 == 0 {
			if nomorkeluaran == nomor_generator {
				result = "GENAP"
				break
			}

		} else {
			if nomorkeluaran == nomor_generator {
				result = "GANJIL"
				break
			}
		}
	}
	return result
}
func _besarkecil(nomorkeluaran string) string {
	nomor_generator := ""
	result := ""
	for i := 0; i <= 99; i++ {
		if i < 10 {
			nomor_generator = "0" + strconv.Itoa(i)
		} else {
			nomor_generator = strconv.Itoa(i)
		}
		if i < 50 {
			if nomorkeluaran == nomor_generator {
				result = "KECIL"
				break
			}

		} else {
			if nomorkeluaran == nomor_generator {
				result = "BESAR"
				break
			}
		}
	}
	return result
}
func _line(nomorkeluaran string) string {
	nomor_generator := ""
	result := ""
	for i := 0; i <= 99; i++ {
		if i < 10 {
			nomor_generator = "0" + strconv.Itoa(i)
		} else {
			nomor_generator = strconv.Itoa(i)
		}
		if i < 19 {
			if nomorkeluaran == nomor_generator {
				result = "LINE1"
				break
			}

		}
		if i > 19 && i < 40 {
			if nomorkeluaran == nomor_generator {
				result = "LINE2"
				break
			}
		}
		if i > 39 && i < 60 {
			if nomorkeluaran == nomor_generator {
				result = "LINE3"
				break
			}
		}
		if i > 59 && i < 80 {
			if nomorkeluaran == nomor_generator {
				result = "LINE4"
				break
			}
		}
		if i > 80 && i < 100 {
			if nomorkeluaran == nomor_generator {
				result = "LINE5"
				break
			}
		}
	}
	return result
}
