package main

import (
	"testing"
	"time"

	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/gorm"
)

func tDate(s string) time.Time {
	v, _ := time.Parse("2006-01-02", s)
	return v
}

func TestDatesOverlap(t *testing.T) {
	cases := []struct {
		name       string
		aStart, aEnd, bStart, bEnd string
		want       bool
	}{
		{"same nights", "2026-11-21", "2026-11-23", "2026-11-20", "2026-11-23", true},
		{"partial overlap", "2026-11-22", "2026-11-25", "2026-11-20", "2026-11-23", true},
		{"touching at checkout", "2026-11-23", "2026-11-25", "2026-11-20", "2026-11-23", false},
		{"disjoint", "2026-11-24", "2026-11-26", "2026-11-20", "2026-11-23", false},
		{"new booking inside old", "2026-11-21", "2026-11-22", "2026-11-20", "2026-11-23", true},
	}
	for _, tc := range cases {
		if got := datesOverlap(tDate(tc.aStart), tDate(tc.aEnd), tDate(tc.bStart), tDate(tc.bEnd)); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestClassifyAvailability(t *testing.T) {
	rooms := []models.Room{
		{Model: gorm.Model{ID: 1}, Number: "101", Status: "available"},
		{Model: gorm.Model{ID: 2}, Number: "102", Status: "available"},
		{Model: gorm.Model{ID: 3}, Number: "103", Status: "cleaning"},
	}
	active := []models.Reservation{
		{Model: gorm.Model{ID: 10}, RoomID: 1, Status: "confirmed", CheckIn: tDate("2026-11-20"), CheckOut: tDate("2026-11-23")},
		{Model: gorm.Model{ID: 11}, RoomID: 1, Status: "canceled", CheckIn: tDate("2026-11-20"), CheckOut: tDate("2026-11-23")},
		{Model: gorm.Model{ID: 12}, RoomID: 2, Status: "checked_in", CheckIn: tDate("2026-11-23"), CheckOut: tDate("2026-11-26")},
	}
	got := classifyAvailability(rooms, active, tDate("2026-11-21"), tDate("2026-11-22"))
	if len(got) != 3 { t.Fatalf("got %d rooms", len(got)) }
	if got[0].Available { t.Error("room 101 must be blocked by reservation 10") }
	if len(got[0].Conflicts) != 1 || got[0].Conflicts[0] != 10 { t.Error("wrong conflicts for room 101") }
	if !got[1].Available { t.Error("room 102 must be free on 21-22") }
	if got[2].Available { t.Error("room 103 is cleaning, must be blocked") }

	got = classifyAvailability(rooms, active, tDate("2026-11-23"), tDate("2026-11-24"))
	if !got[0].Available { t.Error("room 101 must be free after checkout day") }
	if got[1].Available { t.Error("room 102 must be blocked on 23-24") }
}
