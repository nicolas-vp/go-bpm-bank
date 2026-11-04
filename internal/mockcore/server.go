// Package mockcore — самостоятельная учебная АБС с долговечными идемпотентными операциями.
package mockcore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct{ DB *pgxpool.Pool }

func New(ctx context.Context, db *pgxpool.Pool) (*Server, error) {
	_, err := db.Exec(ctx, `CREATE TABLE IF NOT EXISTS mock_records (
	 kind text NOT NULL, key text NOT NULL, hash text NOT NULL, body jsonb NOT NULL,
	 hits integer NOT NULL DEFAULT 1, PRIMARY KEY(kind,key));
	CREATE TABLE IF NOT EXISTS mock_openings (
	 request_id text PRIMARY KEY, hash text NOT NULL, operation_id text UNIQUE NOT NULL,
	 account_id text UNIQUE NOT NULL, scenario text NOT NULL, status text NOT NULL,
	 polls integer NOT NULL DEFAULT 0, registered_at timestamptz);`)
	return &Server{db}, err
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, err error) { write(w, 500, map[string]string{"error": err.Error()}) }
func digest(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// record атомарно сравнивает payload для ключа: повтор — тот же эффект, изменённая команда — конфликт.
func (s *Server) record(ctx context.Context, kind, key string, v any) (int, error) {
	if key == "" {
		return 0, fmt.Errorf("Idempotency-Key обязателен")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return 0, err
	}
	var hits int
	err = s.DB.QueryRow(ctx, `INSERT INTO mock_records(kind,key,hash,body) VALUES($1,$2,$3,$4)
	 ON CONFLICT(kind,key) DO UPDATE SET hits=mock_records.hits+1 WHERE mock_records.hash=excluded.hash RETURNING hits`, kind, key, digest(v), b).Scan(&hits)
	if err == pgx.ErrNoRows {
		return 0, fmt.Errorf("идемпотентный ключ уже использован с другим телом")
	}
	return hits, err
}

type command struct {
	ApplicationID string `json:"applicationId"`
	CustomerID    string `json:"customerId"`
	ContractID    string `json:"contractId"`
	Scenario      string `json:"scenario"`
}
type operation struct {
	OperationID   string     `json:"operationId"`
	AccountID     string     `json:"accountId"`
	AccountNumber string     `json:"accountNumber"`
	Status        string     `json:"status"`
	RegisteredAt  *time.Time `json:"registeredAt"`
}

func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.DB.Ping(r.Context()); err != nil {
			fail(w, err)
			return
		}
		write(w, 200, map[string]string{"status": "ok", "service": "mock-core-banking"})
	})
	m.HandleFunc("POST /v1/checks/{kind}", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			ApplicationID     string `json:"applicationId"`
			Scenario          string `json:"scenario"`
			DocumentsComplete bool   `json:"documentsComplete"`
			TaxResidency      string `json:"taxResidency"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(&in); err != nil {
			write(w, 400, map[string]string{"error": err.Error()})
			return
		}
		status := "VALID"
		switch r.PathValue("kind") {
		case "document":
			if !in.DocumentsComplete {
				status = "INCOMPLETE"
			}
		case "screening":
			status = "CLEAR"
			if in.Scenario == "screening_match" {
				status = "MATCH"
			}
			if in.Scenario == "pep" {
				status = "PEP"
			}
		case "tax":
			if in.TaxResidency == "" || in.Scenario == "tax_refusal" {
				status = "REVIEW"
			}
		default:
			write(w, 404, map[string]string{"error": "неизвестная проверка"})
			return
		}
		write(w, 200, map[string]string{"status": status, "requestId": digest(in)[:16], "checkedAt": time.Now().UTC().Format(time.RFC3339), "source": "synthetic-v1"})
	})
	m.HandleFunc("POST /v1/customers/resolve", func(w http.ResponseWriter, r *http.Request) {
		var in map[string]string
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			write(w, 400, map[string]string{"error": "JSON"})
			return
		}
		if _, err := s.record(r.Context(), "customer", r.Header.Get("Idempotency-Key"), in); err != nil {
			write(w, 409, map[string]string{"error": err.Error()})
			return
		}
		write(w, 200, map[string]string{"customerId": "customer-" + digest(in)[:16]})
	})
	m.HandleFunc("POST /v1/account-openings", s.open)
	m.HandleFunc("GET /v1/account-openings/{id}", s.poll)
	m.HandleFunc("GET /v1/account-openings", func(w http.ResponseWriter, r *http.Request) { s.read(w, r, r.URL.Query().Get("requestId")) })
	m.HandleFunc("POST /v1/effects/{kind}", func(w http.ResponseWriter, r *http.Request) {
		var in map[string]string
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			write(w, 400, map[string]string{"error": "JSON"})
			return
		}
		kind := r.PathValue("kind")
		switch kind {
		case "contract", "archive", "tax-report", "notification":
		default:
			write(w, 404, map[string]string{"error": "неизвестный эффект"})
			return
		}
		hits, err := s.record(r.Context(), kind, r.Header.Get("Idempotency-Key"), in)
		if err != nil {
			write(w, 409, map[string]string{"error": err.Error()})
			return
		}
		if kind == "tax-report" && in["scenario"] == "tax_report_retry" && hits == 1 {
			write(w, 503, map[string]string{"error": "учебная потеря ответа налогового шлюза"})
			return
		}
		write(w, 200, map[string]string{"status": "CONFIRMED", "id": kind + "-" + digest(in)[:16]})
	})
	m.HandleFunc("GET /v1/state", func(w http.ResponseWriter, r *http.Request) {
		var raw []byte
		err := s.DB.QueryRow(r.Context(), `SELECT json_build_object('openings',COALESCE((SELECT json_agg(t) FROM (SELECT request_id,operation_id,account_id,status,scenario,polls,registered_at FROM mock_openings ORDER BY request_id)t),'[]'::json),'effects',COALESCE((SELECT json_agg(t) FROM (SELECT kind,key,hits FROM mock_records ORDER BY kind,key)t),'[]'::json))`).Scan(&raw)
		if err != nil {
			fail(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	})
	return m
}

func (s *Server) open(w http.ResponseWriter, r *http.Request) {
	var in command
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(&in); err != nil {
		write(w, 400, map[string]string{"error": "JSON"})
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if in.ApplicationID == "" || in.CustomerID == "" || in.ContractID == "" || key != in.ApplicationID+":opening:v1" {
		write(w, 400, map[string]string{"error": "неполная команда или неверный ключ"})
		return
	}
	status := "OPENED"
	var registered any = time.Now().UTC()
	if in.Scenario == "core_pending" {
		status = "PENDING"
		registered = nil
	}
	if in.Scenario == "core_reject" {
		status = "REJECTED"
		registered = nil
	}
	h := digest(in)
	var op string
	err := s.DB.QueryRow(r.Context(), `INSERT INTO mock_openings(request_id,hash,operation_id,account_id,scenario,status,registered_at)
	 VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(request_id) DO UPDATE SET request_id=excluded.request_id WHERE mock_openings.hash=excluded.hash RETURNING operation_id`,
		key, h, "op-"+h[:20], "account-"+h[:20], in.Scenario, status, registered).Scan(&op)
	if err == pgx.ErrNoRows {
		write(w, 409, map[string]string{"error": "команда изменилась при том же ключе"})
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	hits, err := s.record(r.Context(), "opening-attempt", key, in)
	if err != nil {
		fail(w, err)
		return
	}
	// Сначала запись счёта коммитится, затем ответ теряется: повтор обязан найти ту же операцию.
	if in.Scenario == "lost_response" && hits == 1 {
		write(w, 503, map[string]string{"error": "ответ потерян после регистрации счёта"})
		return
	}
	s.read(w, r, key)
}

func (s *Server) poll(w http.ResponseWriter, r *http.Request) {
	_, err := s.DB.Exec(r.Context(), `UPDATE mock_openings SET polls=polls+1,
	 status=CASE WHEN status='PENDING' AND polls>=1 THEN 'OPENED' ELSE status END,
	 registered_at=CASE WHEN status='PENDING' AND polls>=1 THEN now() ELSE registered_at END WHERE operation_id=$1`, r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	var key string
	if err := s.DB.QueryRow(r.Context(), `SELECT request_id FROM mock_openings WHERE operation_id=$1`, r.PathValue("id")).Scan(&key); err != nil {
		write(w, 404, map[string]string{"error": "неизвестная операция"})
		return
	}
	s.read(w, r, key)
}

func (s *Server) read(w http.ResponseWriter, r *http.Request, key string) {
	var out operation
	err := s.DB.QueryRow(r.Context(), `SELECT operation_id,account_id,status,registered_at FROM mock_openings WHERE request_id=$1`, key).Scan(&out.OperationID, &out.AccountID, &out.Status, &out.RegisteredAt)
	if err == pgx.ErrNoRows {
		write(w, 404, map[string]string{"error": "неизвестная операция"})
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	// Это заведомо учебный реквизит, не действующий банковский счёт.
	out.AccountNumber = "DEMO-" + out.AccountID[8:20]
	write(w, 200, out)
}
