package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/gorm"
)

// datesOverlap uses half-open intervals [start, end): a booking ending at 14:00
// does not conflict with one starting at 14:00 the same day.
func datesOverlap(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

type roomAvailability struct {
	ID        uint   `json:"id"`
	Number    string `json:"number"`
	Type      string `json:"type"`
	Floor     int    `json:"floor"`
	BaseRate  float64 `json:"base_rate"`
	Status    string `json:"status"`
	Available bool   `json:"available"`
	Conflicts []uint `json:"conflicts"`
}

// classifyAvailability is pure so it can be unit-tested without a database.
// `active` must exclude canceled and checked_out reservations.
func classifyAvailability(rooms []models.Room, active []models.Reservation, start, end time.Time) []roomAvailability {
	byRoom := map[uint][]uint{}
	for _, r := range active {
		if r.Status == "canceled" || r.Status == "checked_out" {
			continue
		}
		if r.RoomID != 0 && datesOverlap(start, end, r.CheckIn, r.CheckOut) {
			byRoom[r.RoomID] = append(byRoom[r.RoomID], r.ID)
		}
	}
	out := make([]roomAvailability, 0, len(rooms))
	for _, room := range rooms {
		conflicts := byRoom[room.ID]
		if conflicts == nil {
			conflicts = []uint{}
		}
		out = append(out, roomAvailability{
			ID: room.ID, Number: room.Number, Type: room.Type, Floor: room.Floor,
			BaseRate: room.BaseRate, Status: room.Status,
			Available: room.Status == "available" && len(conflicts) == 0,
			Conflicts: conflicts,
		})
	}
	return out
}

// GET /api/availability?check_in=YYYY-MM-DD&check_out=YYYY-MM-DD[&hotel_id=N]
func availabilityHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start, err := time.Parse("2006-01-02", c.Query("check_in"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "check_in must be YYYY-MM-DD"})
			return
		}
		end, err := time.Parse("2006-01-02", c.Query("check_out"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "check_out must be YYYY-MM-DD"})
			return
		}
		if !end.After(start) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "check_out must be after check_in"})
			return
		}
		var rooms []models.Room
		q := db.Order("number")
		if h := c.Query("hotel_id"); h != "" {
			q = q.Where("hotel_id = ?", parseID(h))
		}
		if err := q.Find(&rooms).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var active []models.Reservation
		db.Where("status NOT IN ?", []string{"canceled", "checked_out"}).Find(&active)
		c.JSON(http.StatusOK, gin.H{
			"check_in":  start.Format("2006-01-02"),
			"check_out": end.Format("2006-01-02"),
			"rooms":     classifyAvailability(rooms, active, start, end),
		})
	}
}
