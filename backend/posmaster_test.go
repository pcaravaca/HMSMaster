package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIdempotencyKeyStable(t *testing.T) {
	a := idempotencyKey(12, 34)
	b := idempotencyKey(12, 34)
	if a != b || !strings.Contains(a, "c12") || !strings.Contains(a, "r34") {
		t.Fatalf("unstable key: %s vs %s", a, b)
	}
	if !strings.Contains(observationForKey(a), a) {
		t.Fatal("observation must contain the key")
	}
}

func TestFindPOSInvoiceByKeyPaginates(t *testing.T) {
	key := "hms-c99-r1"
	pages := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		page := r.URL.Query().Get("page")
		if r.URL.Query().Get("desde") == "" || r.URL.Query().Get("hasta") == "" {
			t.Error("date window missing")
		}
		if page == "2" {
			fmt.Fprintf(w, `{"facturas":[{"id":"inv-page2","observaciones":"HMSMaster · %s"}],"paginacion":{"total":201}}`, key)
			return
		}
		var sb strings.Builder
		sb.WriteString(`{"facturas":[`)
		for i := 0; i < 200; i++ {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(`{"id":"other","observaciones":"other"}`)
		}
		sb.WriteString(`],"paginacion":{"total":201}}`)
		fmt.Fprint(w, sb.String()) // placeholder
	}))
	defer srv.Close()
	t.Setenv("POSMASTER_API_URL", srv.URL)
	t.Setenv("POSMASTER_TIENDA_ID", "")
	ref, err := findPOSInvoiceByKey(key, time.Now())
	if err != nil || ref != "inv-page2" || pages != 2 {
		t.Fatalf("ref=%q err=%v pages=%d", ref, err, pages)
	}
}

func TestSyncOutcomes(t *testing.T) {
	t.Setenv("POSMASTER_TIENDA_ID", "")
	t.Setenv("POSMASTER_API_TOKEN", "")
	cases := []struct {
		status int
		body   string
		ok     bool
		defin  bool
	}{
		{200, `{"id":"inv-1"}`, true, false},
		{400, `{"error":"bad"}`, false, true},
		{403, `{"error":"forbidden"}`, false, true},
		{500, `{"error":"boom"}`, false, false},
		{429, `{"error":"slow down"}`, false, false},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" || r.URL.Path != "/api/facturas" {
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(tc.status)
			fmt.Fprint(w, tc.body)
		}))
		t.Setenv("POSMASTER_API_URL", srv.URL)
		res := syncChargeToPOSMaster("Minibar", 8500, 3, 7)
		if res.OK != tc.ok || res.Definitive != tc.defin {
			t.Errorf("status %d: got ok=%v definitive=%v", tc.status, res.OK, res.Definitive)
		}
		if tc.ok && res.Ref != "inv-1" {
			t.Errorf("status %d: ref=%q", tc.status, res.Ref)
		}
		srv.Close()
	}
}

func TestFindPOSInvoiceByKey(t *testing.T) {
	key := "hms-c7-r3"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/facturas" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprintf(w, `{"facturas":[{"id":"inv-existing","observaciones":"HMSMaster · %s"}]}`, key)
	}))
	defer srv.Close()
	t.Setenv("POSMASTER_API_URL", srv.URL)
	t.Setenv("POSMASTER_TIENDA_ID", "")
	ref, err := findPOSInvoiceByKey(key, time.Now())
	if err != nil || ref != "inv-existing" {
		t.Fatalf("ref=%q err=%v", ref, err)
	}
	ref, err = findPOSInvoiceByKey("other-key", time.Now())
	if err != nil || ref != "" {
		t.Fatalf("unexpected match ref=%q err=%v", ref, err)
	}
}
