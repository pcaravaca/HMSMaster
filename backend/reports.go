package main

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/gorm"
)

type reportDay struct {
	Date     string  `json:"date"`
	Occupied int     `json:"occupied"`
	Revenue  float64 `json:"revenue"`
}

type reportChannel struct {
	Channel      string  `json:"channel"`
	Reservations int     `json:"reservations"`
	Revenue      float64 `json:"revenue"`
}

type reportSummary struct {
	From                string          `json:"from"`
	To                  string          `json:"to"`
	Days                int             `json:"days"`
	Rooms               int             `json:"rooms"`
	AvailableRoomNights int             `json:"available_room_nights"`
	OccupiedRoomNights  int             `json:"occupied_room_nights"`
	OccupancyPct        float64         `json:"occupancy_pct"`
	RoomRevenue         float64         `json:"room_revenue"`
	ADR                 float64         `json:"adr"`
	RevPAR              float64         `json:"revpar"`
	ByChannel           []reportChannel `json:"by_channel"`
	ByDay               []reportDay     `json:"by_day"`
}

func roundMoney(v float64) float64 { return math.Round(v*100) / 100 }

// activeReportReservation mirrors the reservation validity used by availability.
func activeReportReservation(r models.Reservation) bool {
	// Checked-out stays are history but still count as sold room nights.
	return r.Status != "canceled"
}

// nightsInRange returns the calendar nights of a stay that intersect [from, to].
func nightsInRange(r models.Reservation, from, to time.Time) []time.Time {
	start := r.CheckIn.UTC()
	if start.Before(from) {
		start = from
	}
	// Last night is the day before check-out.
	last := r.CheckOut.UTC().AddDate(0, 0, -1)
	if last.After(to) {
		last = to
	}
	nights := []time.Time{}
	for d := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC); !d.After(last); d = d.AddDate(0, 0, 1) {
		nights = append(nights, d)
	}
	return nights
}

func nightlyPrice(r models.Reservation, day time.Time) float64 {
	var detail []nightQuote
	if r.RateDetail != "" && json.Unmarshal([]byte(r.RateDetail), &detail) == nil {
		key := day.Format(dayLayout)
		for _, q := range detail {
			if q.Date == key {
				return q.Price
			}
		}
	}
	n := nightCount(r.CheckIn, r.CheckOut)
	if n <= 0 || r.TotalAmount <= 0 {
		return 0
	}
	return r.TotalAmount / float64(n)
}

// computeSummary is the pure reporting core: occupancy, ADR, RevPAR and
// revenue by channel for an inclusive date window.
func computeSummary(reservations []models.Reservation, rooms []models.Room, from, to time.Time) reportSummary {
	days := int(to.Sub(from).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	roomIDs := map[uint]bool{}
	for _, room := range rooms {
		roomIDs[room.ID] = true
	}
	byDay := make(map[string]*reportDay, days)
	byChannel := map[string]*reportChannel{}
	occupied, revenue := 0, 0.0
	for _, r := range reservations {
		if !activeReportReservation(r) || !roomIDs[r.RoomID] {
			continue
		}
		nights := nightsInRange(r, from, to)
		if len(nights) == 0 {
			continue
		}
		resRevenue := 0.0
		for _, night := range nights {
			price := nightlyPrice(r, night)
			resRevenue += price
			key := night.Format(dayLayout)
			if byDay[key] == nil {
				byDay[key] = &reportDay{Date: key}
			}
			byDay[key].Occupied++
			byDay[key].Revenue = roundMoney(byDay[key].Revenue + price)
		}
		occupied += len(nights)
		revenue += resRevenue
		ch := r.Channel
		if ch == "" {
			ch = "direct"
		}
		if byChannel[ch] == nil {
			byChannel[ch] = &reportChannel{Channel: ch}
		}
		byChannel[ch].Reservations++
		byChannel[ch].Revenue = roundMoney(byChannel[ch].Revenue + resRevenue)
	}
	available := len(rooms) * days
	summary := reportSummary{
		From: from.Format(dayLayout), To: to.Format(dayLayout), Days: days,
		Rooms: len(rooms), AvailableRoomNights: available, OccupiedRoomNights: occupied,
		RoomRevenue: roundMoney(revenue), ByChannel: []reportChannel{}, ByDay: []reportDay{},
	}
	if available > 0 {
		summary.OccupancyPct = roundMoney(float64(occupied) / float64(available) * 100)
		summary.RevPAR = roundMoney(revenue / float64(available))
	}
	if occupied > 0 {
		summary.ADR = roundMoney(revenue / float64(occupied))
	}
	for d := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC); !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format(dayLayout)
		if byDay[key] == nil {
			byDay[key] = &reportDay{Date: key}
		}
		summary.ByDay = append(summary.ByDay, *byDay[key])
	}
	for _, ch := range byChannel {
		summary.ByChannel = append(summary.ByChannel, *ch)
	}
	return summary
}

func reportSummaryHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, err1 := parseDay(c.Query("from"))
		to, err2 := parseDay(c.Query("to"))
		hotelID := parseID(c.Query("hotel_id"))
		if err1 != nil || err2 != nil || to.Before(from) || hotelID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hotel_id and valid from/to dates (YYYY-MM-DD) are required"})
			return
		}
		var rooms []models.Room
		if err := db.Where("hotel_id = ?", hotelID).Find(&rooms).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var reservations []models.Reservation
		if err := db.Where("hotel_id = ? AND check_in < ? AND check_out >= ?", hotelID, to.AddDate(0, 0, 1), from).Find(&reservations).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, computeSummary(reservations, rooms, from, to))
	}
}
