package engine_test

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

func waitQueue(t *testing.T, pool *pgxpool.Pool, timeout time.Duration) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(timeout)
	for {
		var pending, failed int
		err := pool.QueryRow(ctx, `select count(*) filter (where status = 'pending'), count(*) filter (where status = 'failed') from bpm_queue`).Scan(&pending, &failed)
		if err != nil {
			t.Fatal(err)
		}
		if failed > 0 {
			var lastError string
			_ = pool.QueryRow(ctx, `select kind || ': ' || coalesce(last_error, '') from bpm_queue where status = 'failed' limit 1`).Scan(&lastError)
			t.Fatalf("%d failed tasks, first: %s", failed, lastError)
		}
		if pending == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d tasks still pending", pending)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
