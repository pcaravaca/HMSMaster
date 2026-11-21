package main

import (
	"testing"
	"time"

	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/gorm"
)

func TestComputeSummary(t *testing.T) {
	from, _ := parseDay("2026-11-21")
	to, _ := parseDay("2026-11-22")
	rooms := []models.Room{
		{Model: gorm.Model{ID: 1}, Number: "101"},
		{Model: gorm.Model{ID: 2}, Number: "102"},
	}
	checkIn1, _ := parseTimestamp("2026-11-21T14:00:00Z")
	checkOut1, _ := parseTimestamp("2026-11-23T12:00:00Z")
	checkIn2, _ := parseTimestamp("2026-11-22T14:00:00Z")
	res := []models.Reservation{
		{Model: gorm.Model{ID: 1}, RoomID: 1, CheckIn: checkIn1, CheckOut: checkOut1,
			TotalAmount: 90000, Channel: "airbnb", Status: "checked_out",
			RateDetail: `[{"date":"2026-11-21","price":45000},{"date":"2026-11-22","price":45000}]`},
		{Model: gorm.Model{ID: 2}, RoomID: 2, CheckIn: checkIn2, CheckOut: checkOut1,
			TotalAmount: 50000, Channel: "direct", Status: "confirmed"},
		{Model: gorm.Model{ID: 3}, RoomID: 2, CheckIn: checkIn2, CheckOut: checkOut1,
			TotalAmount: 50000, Channel: "booking", Status: "canceled"},
	}
	s := computeSummary(res, rooms, from, to)
	if s.AvailableRoomNights != 4 || s.OccupiedRoomNights != 3 || s.OccupancyPct != 75 {
		t.Fatalf("occupancy wrong: %+v", s)
	}
	if s.RoomRevenue != 140000 || s.ADR != 46666.67 || s.RevPAR != 35000 {
		t.Fatalf("revenue wrong: %+v", s)
	}
	if len(s.ByDay) != 2 || s.ByDay[0].Occupied != 1 || s.ByDay[0].Revenue != 45000 ||
		s.ByDay[1].Occupied != 2 || s.ByDay[1].Revenue != 95000 {
		t.Fatalf("by_day wrong: %+v", s.ByDay)
	}
	if len(s.ByChannel) != 2 {
		t.Fatalf("by_channel count=%d: %+v", len(s.ByChannel), s.ByChannel)
	}
}

func TestNightlyPriceFallback(t *testing.T) {
	in, _ := time.Parse(time.RFC3339, "2026-12-01T14:00:00Z")
	out, _ := time.Parse(time.RFC3339, "2026-12-03T12:00:00Z")
	r := models.Reservation{CheckIn: in, CheckOut: out, TotalAmount: 90000}
	if got := nightlyPrice(r, in); got != 45000 {
		t.Fatalf("fallback price=%v want 45000", got)
	}
	r.RateDetail = `[{"date":"2026-12-01","price":70000}]`
	if got := nightlyPrice(r, in); got != 70000 {
		t.Fatalf("detail price=%v want 70000", got)
	}
}
