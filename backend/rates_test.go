package main

import (
	"testing"
	"time"

	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/gorm"
)

func testDay(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := parseDay(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestPickRateSpecificity(t *testing.T) {
	rules := []models.RateRule{
		{Model: gorm.Model{ID: 1}, HotelID: 1, RoomID: 7, StartDate: "2026-11-20", EndDate: "2026-11-25", Price: 60000},
		{Model: gorm.Model{ID: 2}, HotelID: 1, RoomID: 0, StartDate: "2026-11-20", EndDate: "2026-11-25", Price: 50000},
	}
	day := testDay(t, "2026-11-22")
	if price, id := pickRate(rules, 7, day, 45000); price != 60000 || id != 1 {
		t.Fatalf("room rule: price=%v id=%d", price, id)
	}
	if price, id := pickRate(rules, 8, day, 45000); price != 50000 || id != 2 {
		t.Fatalf("hotel rule: price=%v id=%d", price, id)
	}
	if price, id := pickRate(rules, 8, testDay(t, "2026-12-01"), 45000); price != 45000 || id != 0 {
		t.Fatalf("outside: price=%v id=%d", price, id)
	}
}

func TestBuildQuoteMixedNights(t *testing.T) {
	rules := []models.RateRule{
		{Model: gorm.Model{ID: 3}, HotelID: 1, RoomID: 0, StartDate: "2026-11-22", EndDate: "2026-11-22", Price: 70000},
	}
	checkIn, _ := parseTimestamp("2026-11-21T14:00:00Z")
	checkOut, _ := parseTimestamp("2026-11-23T12:00:00Z")
	detail, total := buildQuote(rules, checkIn, nightCount(checkIn, checkOut), 5, 45000)
	if len(detail) != 2 || total != 115000 {
		t.Fatalf("detail=%+v total=%v", detail, total)
	}
	if detail[0].Price != 45000 || !detail[0].Base || detail[1].Price != 70000 || detail[1].RateID != 3 {
		t.Fatalf("unexpected nightly pricing: %+v", detail)
	}
}

func TestNightCountAndParsing(t *testing.T) {
	a, _ := parseTimestamp("2026-11-21T14:00:00Z")
	b, _ := parseTimestamp("2026-11-23T12:00:00Z")
	if got := nightCount(a, b); got != 2 {
		t.Fatalf("nights=%d want 2", got)
	}
	if got := nightCount(b, b); got != 1 {
		t.Fatalf("same-day nights=%d want 1", got)
	}
	if _, err := parseTimestamp("not-a-date"); err == nil {
		t.Fatal("invalid date accepted")
	}
}
