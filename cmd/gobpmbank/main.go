// GoBpmBank — самостоятельный учебный экземпляр go-bpm, без стандартных Test*-процессов.
package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolas-vp/go-bpm-bank/engine"
	"github.com/nicolas-vp/go-bpm-bank/httpapi"
	"github.com/nicolas-vp/go-bpm-bank/internal/bankapi"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
	"github.com/nicolas-vp/go-bpm-bank/internal/ui"
	"github.com/nicolas-vp/go-bpm-bank/migrations"
	"github.com/nicolas-vp/go-bpm-bank/queue"
	"github.com/nicolas-vp/go-bpm-bank/webui"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	dsn := os.Getenv("BANK_DSN")
	if dsn == "" {
		return errors.New("BANK_DSN обязателен: используйте отдельную учебную базу")
	}
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = migrations.Up(ctx, db); err != nil {
		return err
	}
	if _, err = db.Exec(ctx, `CREATE TABLE IF NOT EXISTS bank_applications(application_id text PRIMARY KEY, hash text NOT NULL, instance_id text UNIQUE)`); err != nil {
		return err
	}
	mockURL := os.Getenv("MOCK_URL")
	if mockURL == "" {
		mockURL = "http://127.0.0.1:8091"
	}
	svc := &services.Services{MockURL: mockURL, HTTP: &http.Client{Timeout: 5 * time.Second}}
	q := queue.New(db, queue.Options{})
	history := engine.HistoryConfig{ToDB: func() bool { return true }}
	e, err := engine.New(q, engine.Config{KindPrefix: "bank", SystemWorkers: 4, TaskWorkers: 3, History: history, RulesHistory: history}, bpmgen.Processes(svc), bankapi.Queues(svc))
	if err != nil {
		return err
	}
	reader := engine.NewHistoryReader(db, bpmgen.Processes(svc), bpmgen.Decisions)
	mux := http.NewServeMux()
	httpapi.Register(mux, e, reader)
	httpapi.RegisterUserTasks(mux, &bankapi.Tasks{UserTaskService: engine.NewUserTaskService(db, e)})
	httpapi.RegisterOperations(mux, engine.NewInstanceOperations(db, e))
	webui.Register(mux, reader)
	mux.HandleFunc("POST /bank/api/applications", bankapi.Applications(db, e))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			http.Error(w, "database", 503)
			return
		}
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /{$}", ui.Home)
	guard := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Запускать можно только главный процесс через валидируемую идемпотентную команду банка.
		if r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/v1/processes/") {
			http.Error(w, "используйте POST /bank/api/applications", 403)
			return
		}
		w.Header().Set("X-GoBpmBank-Educational-Only", "true")
		mux.ServeHTTP(w, r)
	})
	addr := os.Getenv("BANK_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8090"
	}
	server := &http.Server{Addr: addr, Handler: guard, ReadHeaderTimeout: 5 * time.Second}
	q.Start(ctx)
	defer q.Stop()
	errs := make(chan error, 1)
	go func() {
		log.Printf("GoBpmBank: http://%s — только учебные цели", addr)
		errs <- server.ListenAndServe()
	}()
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
