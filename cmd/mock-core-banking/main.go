// Отдельный учебный сервис АБС; настоящих банковских операций не выполняет.
package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolas-vp/go-bpm-bank/internal/mockcore"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	dsn := os.Getenv("MOCK_DSN")
	if dsn == "" {
		log.Fatal("MOCK_DSN обязателен: отдельная база мок-АБС")
	}
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	s, err := mockcore.New(context.Background(), db)
	if err != nil {
		log.Fatal(err)
	}
	addr := os.Getenv("MOCK_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8091"
	}
	log.Printf("GoBpmBank mock-core-banking: %s — только учебные данные", addr)
	server := &http.Server{Addr: addr, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}
