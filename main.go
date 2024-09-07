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
	"time"

	"bitbucket.org/isbtotogroup/wigo_engine_invoice/db"
	"bitbucket.org/isbtotogroup/wigo_engine_invoice/helpers"
	"bitbucket.org/isbtotogroup/wigo_engine_invoice/models"
	"github.com/buger/jsonparser"
	"github.com/go-redis/redis"
	"github.com/joho/godotenv"
	"github.com/nleeper/goment"
)

const invoice_client_redis = "CLIENT:LISTINVOICE"
const invoice_result_redis = "CLIENT:RESULT"
const invoice_agen_redis = "AGEN:12D30S:LISTINVOICE"

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
	flag_detail := false
	keyredis := strings.ToLower(idcompany) + ":12D30S:invoice_" + invoice
	resultRD_invoice, flag_invoice := helpers.GetRedis(keyredis)
	_, tbl_trx_transaksi, tbl_trx_transaksidetail, tbl_trx_transaksimember, _ := models.Get_mappingdatabase(idcompany)
	if !flag_invoice {
		fmt.Println("READ INVOICE DATABASE")

		con := db.CreateCon()
		ctx := context.Background()

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
				// win = int(float64(bet_db) * multiplier_db)
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

			key_redis_invoice_client := strings.ToLower(idcompany) + ":" + invoice_client_redis + "_" + strings.ToLower(username_client_db)
			val_invoice_client := helpers.DeleteRedis(key_redis_invoice_client)
			fmt.Println("")
			fmt.Printf("Redis Delete INVOICE : %d - %s \r", val_invoice_client, key_redis_invoice_client)
			fmt.Println("")
		}
		defer row.Close()

	} else {
		fmt.Println("READ INVOICE REDIS")

		jsonredis := []byte(resultRD_invoice)
		recordlistbet_RD, _, _, _ := jsonparser.Get(jsonredis, "listbet")

		jsonparser.ArrayEach(recordlistbet_RD, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			client_id, _ := jsonparser.GetString(value, "client_id")
			client_username, _ := jsonparser.GetString(value, "client_username")
			client_tipebet, _ := jsonparser.GetString(value, "client_tipebet")
			client_nomor, _ := jsonparser.GetString(value, "client_nomor")
			client_bet, _ := jsonparser.GetInt(value, "client_bet")
			client_multiplier, _ := jsonparser.GetFloat(value, "client_multiplier")

			status_client := _rumuswigo(client_tipebet, client_nomor, result)
			win := 0
			if status_client == "WIN" {
				win = int(client_bet) + int(float64(client_bet)*client_multiplier)
				// win = int(float64(client_bet) * client_multiplier)
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
				"SYSTEM", tglnow.Format("YYYY-MM-DD HH:mm:ss"), client_id)

			if !flag_update_detail {
				fmt.Println(msg_update_detail)
			}
			flag_detail = true

			key_redis_invoice_client := strings.ToLower(idcompany) + ":" + invoice_client_redis + "_" + strings.ToLower(client_username)
			val_invoice_client := helpers.DeleteRedis(key_redis_invoice_client)
			fmt.Println("")
			fmt.Printf("Redis Delete INVOICE : %d - %s \r", val_invoice_client, key_redis_invoice_client)
			fmt.Println("")

		})

	}
	if flag_detail {
		// UPDATE TABLE MEMBER
		con := db.CreateCon()
		ctx := context.Background()

		sql_select_detail := `SELECT 
					playerinvoice, username_client, 
					sum(bet) as payout,
					sum(win) as win, 
					CASE
						WHEN sum(bet) = sum(win) THEN 'DRAW'
						WHEN sum(bet) < sum(win) THEN 'WIN'
						ELSE 'LOSE'
					END AS status 
					FROM ` + tbl_trx_transaksidetail + `  
					WHERE idtransaksi='` + invoice + `'
					AND status_transaksidetail != 'RUNNING'  
					group by playerinvoice, username_client 
			`

		row, err := con.QueryContext(ctx, sql_select_detail)
		helpers.ErrorCheck(err)
		for row.Next() {
			var (
				payout_db, win_db                                int
				playerinvoice_db, nusername_client_db, status_db string
			)

			err = row.Scan(&playerinvoice_db, &nusername_client_db, &payout_db, &win_db, &status_db)
			helpers.ErrorCheck(err)

			sql_insert := `
				insert into
				` + tbl_trx_transaksimember + ` (
					idtransaksimember, idtransaksi , username_client, 
					playerinvoice, totalbet, totalwin, status_transaksimember,  
					create_transaksimember, createdate_transaksimember  
				) values (
					$1, $2, $3, 
					$4, $5, $6, $7,   
					$8, $9   
				)
			`

			field_column := tbl_trx_transaksimember + tglnow.Format("YYYY") + tglnow.Format("MM") + tglnow.Format("DD")
			idrecord_counter := models.Get_counter(field_column)
			idrecrodmember_value := tglnow.Format("YY") + tglnow.Format("MM") + tglnow.Format("DD") + tglnow.Format("HH") + strconv.Itoa(idrecord_counter)
			date_transaksi := tglnow.Format("YYYY-MM-DD HH:mm:ss")

			flag_insert, msg_insert := models.Exec_SQL(sql_insert, tbl_trx_transaksimember, "INSERT",
				idrecrodmember_value, invoice, nusername_client_db,
				playerinvoice_db, payout_db, win_db, status_db,
				"SYSTEM", date_transaksi)

			if !flag_insert {
				fmt.Println(msg_insert)
			}

		}
		defer row.Close()

		// UPDATE PARENT
		total_bet, total_win, total_pair, total_invoice, total_member := _GetTotalBetWin_Transaksi(tbl_trx_transaksidetail, invoice)
		sql_update_parent := `
				UPDATE 
				` + tbl_trx_transaksi + `  
				SET total_bet=$1, total_win=$2, total_member=$3, 
				total_pairs=$4, total_invoice=$5,             
				update_transaksi=$6, updatedate_transaksi=$7             
				WHERE idtransaksi=$8         
			`
		flag_update_parent, msg_update_parent := models.Exec_SQL(sql_update_parent, tbl_trx_transaksi, "UPDATE",
			total_bet, total_win, total_member, total_pair, total_invoice,
			"SYSTEM", tglnow.Format("YYYY-MM-DD HH:mm:ss"), invoice)

		if !flag_update_parent {
			fmt.Println(msg_update_parent)

		} else {
			// flag_compile = true
			msg = "Success - Update Paret - " + invoice
		}

		type Invoicemonth struct {
			Totalbet int `json:"totalbet"`
			Totalwin int `json:"totalwin"`
		}
		var objinvoicedaily Invoicemonth
		var objinvoicemonth Invoicemonth

		dayendmonth := helpers.GetEndRangeDate(tglnow.Format("MM"))
		tglstart := tglnow.Format("YYYY-MM-") + "01 00:00:00"
		tglend := tglnow.Format("YYYY-MM-") + dayendmonth + " 23:59:59"
		tglstart_daily := tglnow.Format("YYYY-MM-DD") + " 00:00:00"
		tglend_daily := tglnow.Format("YYYY-MM-DD") + " 23:59:59"

		tglstartdaily_redis := tglnow.Format("YYYYMMDD") + "000000"
		tglenddaily_redis := tglnow.Format("YYYYMMDD") + "235959"
		tglstart_redis := tglnow.Format("YYYYMM") + "01000000"
		tglend_redis := tglnow.Format("YYYYMM") + dayendmonth + "235959"

		keyredis_invoicedaily := strings.ToLower(idcompany) + ":12D30S:invoicedaily_" + tglstartdaily_redis + tglenddaily_redis
		keyredis_invoicemonth := strings.ToLower(idcompany) + ":12D30S:invoicemonth_" + tglstart_redis + tglend_redis
		resultRD_invoicedaily, flag_invoicedaily := helpers.GetRedis(keyredis_invoicedaily)
		resultRD_invoicemonth, flag_invoicemonth := helpers.GetRedis(keyredis_invoicemonth)
		if !flag_invoicemonth {
			fmt.Println("INVOICE MONTH DATABASE")
			totalbet_DB, totalwin_DB := _GetTotalBet_ByDate(tbl_trx_transaksi, tglstart, tglend)

			objinvoicemonth.Totalbet = totalbet_DB
			objinvoicemonth.Totalwin = totalwin_DB

			helpers.SetRedis(keyredis_invoicemonth, objinvoicemonth, 0)
		} else {
			fmt.Println("INVOICE MONTH CACHE")
			jsonredis := []byte(resultRD_invoicemonth)
			totalbet_RD, _ := jsonparser.GetInt(jsonredis, "totalbet")
			totalwin_RD, _ := jsonparser.GetInt(jsonredis, "totalwin")

			totalwinnew_month := total_win + int(totalwin_RD)

			objinvoicemonth.Totalbet = int(totalbet_RD)
			objinvoicemonth.Totalwin = int(totalwinnew_month)

			helpers.SetRedis(keyredis_invoicemonth, objinvoicemonth, 0)
		}
		if !flag_invoicedaily {
			fmt.Println("INVOICE DAILY DATABASE")
			totalbet_DB, totalwin_DB := _GetTotalBet_ByDate(tbl_trx_transaksi, tglstart_daily, tglend_daily)

			objinvoicedaily.Totalbet = totalbet_DB
			objinvoicedaily.Totalwin = totalwin_DB

			helpers.SetRedis(keyredis_invoicedaily, objinvoicedaily, 2880*time.Minute)
		} else {
			fmt.Println("INVOICE DAILY CACHE")
			jsonredis := []byte(resultRD_invoicedaily)
			totalbet_RD, _ := jsonparser.GetInt(jsonredis, "totalbet")
			totalwin_RD, _ := jsonparser.GetInt(jsonredis, "totalwin")

			totalwinnew_month := total_win + int(totalwin_RD)

			objinvoicedaily.Totalbet = int(totalbet_RD)
			objinvoicedaily.Totalwin = int(totalwinnew_month)

			helpers.SetRedis(keyredis_invoicedaily, objinvoicedaily, 2880*time.Minute)
		}
	}

	key_redis_result := strings.ToLower(idcompany) + ":" + invoice_result_redis
	val_result := helpers.DeleteRedis(key_redis_result)
	fmt.Println("")
	fmt.Printf("Redis Delete RESULT : %d - %s \n", val_result, key_redis_result)
	fmt.Println("")
	for i := 0; i <= 1000; i = i + 250 {
		//LISTINVOICE_2D30S_AGEN_nuke_0_
		key_redis_ageninvoice := strings.ToLower(idcompany) + ":" + invoice_agen_redis + "_" + strconv.Itoa(i) + "_"
		val_result := helpers.DeleteRedis(key_redis_ageninvoice)
		fmt.Printf("Redis Delete AGEN INVOICE : %d - %s \n", val_result, key_redis_ageninvoice)
	}

	// strings.ToLower(client_company) + ":" + Fieldtransaksi2d30s_home_redis + "_DETAIL_" + client.Transaksidetail2D30S_invoice + "_" + client.Transaksidetail2D30S_status
	// key_redis_detail := "LISTINVOICE_2D30S_AGEN_nuke_DETAIL_240312231346_WIN"
	key_redis_detail_win := strings.ToLower(idcompany) + ":" + invoice_agen_redis + "_DETAIL_" + invoice + "_WIN"
	key_redis_detail_lose := strings.ToLower(idcompany) + ":" + invoice_agen_redis + "_DETAIL_" + invoice + "_LOSE"
	key_redis_detail_running := strings.ToLower(idcompany) + ":" + invoice_agen_redis + "_DETAIL_" + invoice + "_RUNNING"
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
func _GetTotalBet_ByDate(table, startdate, enddate string) (int, int) {
	con := db.CreateCon()
	ctx := context.Background()
	totalbet := 0
	totalwin := 0
	sql_select := ""
	sql_select += "SELECT "
	sql_select += "COALESCE(SUM(total_bet),0) AS totalbet,  COALESCE(sum(total_win),0) as totalwin "
	sql_select += "FROM " + table + " "
	sql_select += "WHERE createdate_transaksi >='" + startdate + "'   "
	sql_select += "AND createdate_transaksi <='" + enddate + "'   "

	row := con.QueryRowContext(ctx, sql_select)
	switch e := row.Scan(&totalbet, &totalwin); e {
	case sql.ErrNoRows:
	case nil:
	default:
		helpers.ErrorCheck(e)
	}

	return totalbet, totalwin
}
func _GetTotalBetWin_Transaksi(table, idtransaksi string) (int, int, int, int, int) {
	con := db.CreateCon()
	ctx := context.Background()
	total_bet := 0
	total_win := 0
	total_pair := 0
	total_invoice := 0
	total_member := 0
	sql_select := ""
	sql_select += "SELECT "
	sql_select += "SUM(bet) AS total_bet, SUM(win) AS total_win,   "
	sql_select += "COUNT(idtransaksidetail) AS total_pair, COUNT(DISTINCT(playerinvoice)) AS total_invoice,   "
	sql_select += "COUNT(DISTINCT(username_client)) AS total_member   "
	sql_select += "FROM " + table + " "
	sql_select += "WHERE idtransaksi='" + idtransaksi + "'   "
	sql_select += "AND status_transaksidetail !='RUNNING'   "

	row := con.QueryRowContext(ctx, sql_select)
	switch e := row.Scan(&total_bet, &total_win, &total_pair, &total_invoice, &total_member); e {
	case sql.ErrNoRows:
	case nil:
	default:
		helpers.ErrorCheck(e)
	}

	return total_bet, total_win, total_pair, total_invoice, total_member
}

func _rumuswigo(tipebet, nomorclient, nomorkeluaran string) string {
	result := "LOSE"

	result_redblack, result_gangen, result_besarkecil, result_line, result_zona := _nomorresult(nomorkeluaran)
	switch tipebet {
	case "ANGKA":
		if nomorclient == nomorkeluaran {
			result = "WIN"
		}
	case "REDBLACK":
		if nomorclient == result_redblack {
			result = "WIN"
		}
		if nomorclient == result_gangen {
			result = "WIN"
		}
		if nomorclient == result_besarkecil {
			result = "WIN"
		}
	case "LINE":
		if nomorclient == result_line {
			result = "WIN"
		}
	case "ZONA":
		if nomorclient == result_zona {
			result = "WIN"
		}
	}

	return result
}

func _nomorresult(nomoresult string) (string, string, string, string, string) {
	type nomor_result_data struct {
		nomor_id         string
		nomor_value      string
		nomor_flag       bool
		nomor_css        string
		nomor_gangen     string
		nomor_besarkecil string
		nomor_line       string
		nomor_zona       string
		nomor_redblack   string
	}

	var cards = []nomor_result_data{
		{nomor_id: "01", nomor_value: "01", nomor_zona: "ZONA_A", nomor_flag: false, nomor_css: "btn btn-error", nomor_gangen: "GANJIL", nomor_besarkecil: "KECIL", nomor_line: "LINE1", nomor_redblack: "RED"},
		{nomor_id: "04", nomor_value: "04", nomor_zona: "ZONA_A", nomor_flag: false, nomor_css: "btn", nomor_gangen: "GENAP", nomor_besarkecil: "KECIL", nomor_line: "LINE2", nomor_redblack: "BLACK"},
		{nomor_id: "07", nomor_value: "07", nomor_zona: "ZONA_A", nomor_flag: false, nomor_css: "btn btn-error", nomor_gangen: "GANJIL", nomor_besarkecil: "BESAR", nomor_line: "LINE3", nomor_redblack: "RED"},
		{nomor_id: "10", nomor_value: "10", nomor_zona: "ZONA_A", nomor_flag: false, nomor_css: "btn", nomor_gangen: "GENAP", nomor_besarkecil: "BESAR", nomor_line: "LINE4", nomor_redblack: "BLACK"},
		{nomor_id: "02", nomor_value: "02", nomor_zona: "ZONA_B", nomor_flag: false, nomor_css: "btn btn-error", nomor_gangen: "GENAP", nomor_besarkecil: "KECIL", nomor_line: "LINE1", nomor_redblack: "RED"},
		{nomor_id: "05", nomor_value: "05", nomor_zona: "ZONA_B", nomor_flag: false, nomor_css: "btn", nomor_gangen: "GANJIL", nomor_besarkecil: "KECIL", nomor_line: "LINE2", nomor_redblack: "BLACK"},
		{nomor_id: "08", nomor_value: "08", nomor_zona: "ZONA_B", nomor_flag: false, nomor_css: "btn btn-error", nomor_gangen: "GENAP", nomor_besarkecil: "BESAR", nomor_line: "LINE3", nomor_redblack: "RED"},
		{nomor_id: "11", nomor_value: "11", nomor_zona: "ZONA_B", nomor_flag: false, nomor_css: "btn", nomor_gangen: "GANJIL", nomor_besarkecil: "BESAR", nomor_line: "LINE4", nomor_redblack: "BLACK"},
		{nomor_id: "03", nomor_value: "03", nomor_zona: "ZONA_C", nomor_flag: false, nomor_css: "btn btn-error", nomor_gangen: "GANJIL", nomor_besarkecil: "KECIL", nomor_line: "LINE2", nomor_redblack: "RED"},
		{nomor_id: "06", nomor_value: "06", nomor_zona: "ZONA_C", nomor_flag: false, nomor_css: "btn", nomor_gangen: "GENAP", nomor_besarkecil: "KECIL", nomor_line: "LINE2", nomor_redblack: "BLACK"},
		{nomor_id: "09", nomor_value: "09", nomor_zona: "ZONA_C", nomor_flag: false, nomor_css: "btn btn-error", nomor_gangen: "GANJIL", nomor_besarkecil: "BESAR", nomor_line: "LINE3", nomor_redblack: "RED"},
		{nomor_id: "12", nomor_value: "12", nomor_zona: "ZONA_C", nomor_flag: false, nomor_css: "btn", nomor_gangen: "GENAP", nomor_besarkecil: "BESAR", nomor_line: "LINE4", nomor_redblack: "BLACK"},
		{nomor_id: "JP", nomor_value: "JP", nomor_flag: false, nomor_css: "btn", nomor_gangen: "GANJIL", nomor_besarkecil: "BESAR", nomor_line: "LINE3", nomor_redblack: "RED"}}

	result_redblack := ""
	result_gangen := ""
	result_besarkecil := ""
	result_line := ""
	result_zona := ""
	for i := 0; i < len(cards); i++ {
		if cards[i].nomor_id == nomoresult {
			result_redblack = cards[i].nomor_redblack
			result_gangen = cards[i].nomor_gangen
			result_besarkecil = cards[i].nomor_besarkecil
			result_line = cards[i].nomor_line
			result_zona = cards[i].nomor_zona
		}
	}
	return result_redblack, result_gangen, result_besarkecil, result_line, result_zona
}
