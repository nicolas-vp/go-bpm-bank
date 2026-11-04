package mockcore

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http/httptest"
	"os"
	"testing"
)

// Проверяет долговечность операции при пересоздании сервиса, а не только кеш в памяти.
func TestOpeningSurvivesServerRecreation(t *testing.T) {
	dsn := os.Getenv("MOCK_TEST_DSN")
	if dsn == "" {
		t.Skip("MOCK_TEST_DSN не задан")
	}
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id := uuid.NewString()
	key := id + ":opening:v1"
	in := command{ApplicationID: id, CustomerID: "DEMO-CUSTOMER", ContractID: "DEMO-CONTRACT", Scenario: "lost_response"}
	call := func(server *Server, command command) *httptest.ResponseRecorder {
		b, _ := json.Marshal(command)
		req := httptest.NewRequest("POST", "/v1/account-openings", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		return w
	}
	first, err := New(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if w := call(first, in); w.Code != 503 {
		t.Fatalf("нужна потеря ответа после коммита: %d %s", w.Code, w.Body.String())
	}
	second, err := New(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	w := call(second, in)
	if w.Code != 200 {
		t.Fatalf("повтор после пересоздания: %d %s", w.Code, w.Body.String())
	}
	var result operation
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "OPENED" || result.RegisteredAt == nil {
		t.Fatalf("счёт не зарегистрирован: %+v", result)
	}
	var count int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM mock_openings WHERE request_id=$1`, key).Scan(&count); err != nil || count != 1 {
		t.Fatalf("дубликат операции: %d %v", count, err)
	}
	in.ContractID = "CHANGED"
	if w := call(second, in); w.Code != 409 {
		t.Fatalf("изменённая команда должна конфликтовать: %d", w.Code)
	}
}
