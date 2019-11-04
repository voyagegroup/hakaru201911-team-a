package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"os"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
	tracer.Start(tracer.WithServiceName("test-go"))
	defer tracer.Stop()

	dataSourceName := os.Getenv("HAKARU_DATASOURCENAME")
	if dataSourceName == "" {
		dataSourceName = "root:password@tcp(127.0.0.1:13306)/hakaru"
	}

	maxConnections := 66
	numInstance := 15

	sqltrace.Register("mysql", &mysql.MySQLDriver{}, sqltrace.WithServiceName("my-db"))
	db, err := sqltrace.Open("mysql", dataSourceName)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	db.SetMaxOpenConns(maxConnections / numInstance)

	hakaruHandler := func(w http.ResponseWriter, r *http.Request) {
		stmt, e := db.Prepare("INSERT INTO eventlog(at, name, value) values(NOW(), ?, ?)")
		if e != nil {
			panic(e.Error())
		}

		defer stmt.Close()

		name := r.URL.Query().Get("name")
		value := r.URL.Query().Get("value")

		_, _ = stmt.Exec(name, value)

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

	//mux := httptrace.NewServeMux()
	//mux.HandleFunc("/hakaru", hakaruHandler)
	http.HandleFunc("/hakaru", hakaruHandler)

	http.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "Set keep alive")
	})

	// start server
	s := &http.Server{
		Addr:           ":8081",
		Handler:        nil,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    80 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.SetKeepAlivesEnabled(true)

	log.Fatal(s.ListenAndServe())
	//if err := http.ListenAndServe(":8081", mux); err != nil {
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
