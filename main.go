package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"os"

    "github.com/go-sql-driver/mysql"
	"github.com/carlescere/scheduler"
	_ "github.com/go-sql-driver/mysql"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
    sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type EventLog struct {
	At    time.Time
	Name  string
	Value string
}

func insertChunk(valueStrings []string, valueArgs [](interface{}), db *sql.DB) {
	stmt := fmt.Sprintf("INSERT INTO eventlog(at, name, value) VALUES %s", strings.Join(valueStrings, ","))
	_, e := db.Exec(stmt, valueArgs...)
	if e != nil {
		panic(e.Error())
	}
}

func insert(resc chan EventLog, db *sql.DB) {
	const chunkSize = 1000

	valueStrings := []string{}
	valueArgs := [](interface{}){}

LOOP:
	for {
		select {
		case eventLog, ok := <-resc:
			if ok {
				valueStrings = append(valueStrings, "(?, ?, ?)")
				valueArgs = append(valueArgs, fmt.Sprintf("%s", eventLog.At))
				valueArgs = append(valueArgs, eventLog.Name)
				valueArgs = append(valueArgs, eventLog.Value)
				if len(valueStrings) >= chunkSize {
					insertChunk(valueStrings, valueArgs, db)
					valueStrings = nil
					valueArgs = nil
				}
			} else {
				panic("resc is closed!!!")
			}
		default:
			break LOOP
		}
	}

	if len(valueStrings) == 0 {
		return
	}

	insertChunk(valueStrings, valueArgs, db)
}

func main() {
	tracer.Start(tracer.WithServiceName("test-go"))
	defer tracer.Stop()

	dataSourceName := os.Getenv("HAKARU_DATASOURCENAME")
	if dataSourceName == "" {
		dataSourceName = "root:password@tcp(127.0.0.1:13306)/hakaru"
	}

    sqltrace.Register("mysql", &mysql.MySQLDriver{}, sqltrace.WithServiceName("my-db"))
    db, err := sqltrace.Open("mysql", dataSourceName)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	maxConnections := 66
	numInstance := 15
	db.SetMaxOpenConns(maxConnections / numInstance)

	resc := make(chan EventLog, 200000)

	_, e := scheduler.Every(10).Seconds().NotImmediately().Run(func() {
		insert(resc, db)
	})

	if e != nil {
		panic(err.Error())
	}

	jst, e := time.LoadLocation("Asia/Tokyo")

	if e != nil {
		panic(e.Error())
	}

	hakaruHandler := func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		value := r.URL.Query().Get("value")

		now := time.Now().In(jst)

		resc <- EventLog{
			At:    now,
			Name:  name,
			Value: value,
		}

		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
	}

	mux := httptrace.NewServeMux()
	mux.HandleFunc("/hakaru", hakaruHandler)
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

	// start server
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
