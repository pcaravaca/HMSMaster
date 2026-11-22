package main

import (
	"strings"
	"testing"
)

func TestEmbeddedBaselineMigration(t *testing.T) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil || len(entries) == 0 {
		t.Fatalf("no embedded migrations: %v", err)
	}
	body, err := migrationFiles.ReadFile("migrations/0001_baseline.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"hotels", "rooms", "guests", "reservations", "folio_charges", "rate_rules", "users", "channel_bookings", "housekeeping_tasks"} {
		if !strings.Contains(string(body), "CREATE TABLE IF NOT EXISTS public."+table) {
			t.Fatalf("missing table %s in baseline", table)
		}
	}
	if strings.Contains(string(body), "\n    id bigint NOT NULL,") {
		t.Fatal("baseline must use bigserial primary keys")
	}
	if !strings.Contains(string(body), "schema_migrations") && !strings.Contains(string(body), "IF NOT EXISTS") {
		t.Fatal("baseline must be idempotent")
	}
}
