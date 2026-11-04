package bankapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolas-vp/go-bpm-bank/engine"
	"net/http"
)

var Scenarios = map[string]bool{"happy": true, "incomplete": true, "screening_match": true, "pep": true, "tax_refusal": true, "lost_response": true, "core_pending": true, "core_reject": true, "tax_report_retry": true, "withdrawn": true}

func Applications(db *pgxpool.Pool, e *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			ApplicationID string `json:"applicationId"`
			Scenario      string `json:"scenario"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&in); err != nil {
			http.Error(w, "только applicationId и scenario", 400)
			return
		}
		if _, err := uuid.Parse(in.ApplicationID); err != nil {
			http.Error(w, "applicationId должен быть UUID", 400)
			return
		}
		if !Scenarios[in.Scenario] {
			http.Error(w, "неизвестный учебный сценарий", 400)
			return
		}
		b, _ := json.Marshal(in)
		h := sha256.Sum256(b)
		hash := hex.EncodeToString(h[:])
		tx, err := db.Begin(r.Context())
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer tx.Rollback(r.Context())
		// Резервирование ключа и старт движка атомарны в одной БД.
		var storedHash string
		var instance string
		err = tx.QueryRow(r.Context(), `INSERT INTO bank_applications(application_id,hash) VALUES($1,$2)
		 ON CONFLICT(application_id) DO UPDATE SET application_id=excluded.application_id RETURNING hash,COALESCE(instance_id,'')`, in.ApplicationID, hash).Scan(&storedHash, &instance)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if storedHash != hash {
			http.Error(w, "applicationId уже связан с другой командой", 409)
			return
		}
		status := 200
		if instance == "" {
			id, err := e.StartProcessTx(r.Context(), tx, "RetailAccountOpening", map[string]any{"applicationId": in.ApplicationID, "scenario": in.Scenario}, engine.WithBusinessKey(in.ApplicationID))
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			instance = id.String()
			status = 201
			if _, err := tx.Exec(r.Context(), `UPDATE bank_applications SET instance_id=$2 WHERE application_id=$1`, in.ApplicationID, instance); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
		if err := tx.Commit(r.Context()); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = fmt.Fprintf(w, `{"id":%q,"applicationId":%q,"educationalOnly":true}`, instance, in.ApplicationID)
	}
}
