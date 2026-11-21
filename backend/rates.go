package main

import (
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/gorm"
)

const dayLayout = "2006-01-02"

type nightQuote struct {
	Date    string  `json:"date"`
	Price   float64 `json:"price"`
	RateID  uint    `json:"rate_id,omitempty"`
	Base    bool    `json:"base,omitempty"`
}

// parseDay accepts an ISO date and normalizes it to UTC midnight.
func parseDay(s string) (time.Time, error) {
	return time.ParseInLocation(dayLayout, s, time.UTC)
}

func ruleApplies(r models.RateRule, roomID uint, day time.Time) bool {
	if r.RoomID != 0 && r.RoomID != roomID {
		return false
	}
	start, err := parseDay(r.StartDate)
	if err != nil || day.Before(start) {
		return false
	}
	end, err := parseDay(r.EndDate)
	if err != nil || day.After(end) {
		return false
	}
	return true
}

// pickRate chooses the most specific applicable rule. Rules are expected in
// descending ID order, so the newest rule wins a tie.
func pickRate(rules []models.RateRule, roomID uint, day time.Time, base float64) (float64, uint) {
	price, id := base, uint(0)
	bestSpecificity := -1
	for _, r := range rules {
		if !ruleApplies(r, roomID, day) || r.Price <= 0 {
			continue
		}
		specificity := 0
		if r.RoomID != 0 {
			specificity = 1
		}
		if specificity > bestSpecificity {
			price, id, bestSpecificity = r.Price, r.ID, specificity
		}
	}
	return price, id
}

// parseTimestamp accepts both full ISO timestamps and plain YYYY-MM-DD dates.
func parseTimestamp(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return parseDay(s)
}

func nightCount(checkIn, checkOut time.Time) int {
	n := int(math.Round(checkOut.Sub(checkIn).Hours() / 24))
	if n < 1 {
		return 1
	}
	return n
}

// quoteStay resolves the rate for every night of a stay. Check-in times are
// normalized to the UTC calendar day so a 14:00 arrival uses that day's rate.
func quoteStay(db *gorm.DB, hotelID, roomID uint, checkIn, checkOut time.Time, base float64) ([]nightQuote, float64, error) {
	nights := nightCount(checkIn, checkOut)
	first := checkIn.UTC().Format(dayLayout)
	last := checkIn.UTC().AddDate(0, 0, nights-1).Format(dayLayout)
	var rules []models.RateRule
	if err := db.Where("hotel_id = ? AND (room_id = 0 OR room_id = ?) AND start_date <= ? AND end_date >= ?",
		hotelID, roomID, last, first).Order("id DESC").Find(&rules).Error; err != nil {
		return nil, 0, err
	}
	detail, total := buildQuote(rules, checkIn, nights, roomID, base)
	return detail, total, nil
}

// buildQuote is the pure pricing core, testable without a database.
func buildQuote(rules []models.RateRule, checkIn time.Time, nights int, roomID uint, base float64) ([]nightQuote, float64) {
	detail := make([]nightQuote, 0, nights)
	total := 0.0
	for i := 0; i < nights; i++ {
		day := time.Date(checkIn.UTC().Year(), checkIn.UTC().Month(), checkIn.UTC().Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)
		price, ruleID := pickRate(rules, roomID, day, base)
		detail = append(detail, nightQuote{Date: day.Format(dayLayout), Price: price, RateID: ruleID, Base: ruleID == 0})
		total += price
	}
	return detail, math.Round(total*100) / 100
}

func listRates(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := db.Model(&models.RateRule{}).Order("id DESC")
		if hotelID := parseID(c.Query("hotel_id")); hotelID != 0 {
			q = q.Where("hotel_id = ?", hotelID)
		}
		if roomID := parseID(c.Query("room_id")); roomID != 0 {
			q = q.Where("room_id = ?", roomID)
		}
		var rules []models.RateRule
		if err := q.Find(&rules).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, rules)
	}
}

func createRate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r models.RateRule
		if err := c.ShouldBindJSON(&r); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		start, err1 := parseDay(r.StartDate)
		end, err2 := parseDay(r.EndDate)
		if r.HotelID == 0 || r.Price <= 0 || err1 != nil || err2 != nil || end.Before(start) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hotel_id, positive price and valid start_date <= end_date are required"})
			return
		}
		if r.RoomID != 0 {
			var room models.Room
			if err := db.First(&room, r.RoomID).Error; err != nil || room.HotelID != r.HotelID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "room not found in this hotel"})
				return
			}
		}
		if err := db.Create(&r).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, r)
	}
}

func deleteRate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.Delete(&models.RateRule{}, parseID(c.Param("id"))).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

func quoteRate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		hotelID := parseID(c.Query("hotel_id"))
		roomID := parseID(c.Query("room_id"))
		checkIn, err1 := parseTimestamp(c.Query("check_in"))
		checkOut, err2 := parseTimestamp(c.Query("check_out"))
		if hotelID == 0 || roomID == 0 || err1 != nil || err2 != nil || !checkOut.After(checkIn) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hotel_id, room_id and valid check_in/check_out are required"})
			return
		}
		var room models.Room
		if err := db.First(&room, roomID).Error; err != nil || room.HotelID != hotelID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "room not found in this hotel"})
			return
		}
		detail, total, err := quoteStay(db, hotelID, roomID, checkIn, checkOut, room.BaseRate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"hotel_id": hotelID, "room_id": roomID, "base_rate": room.BaseRate,
			"nights": len(detail), "total": total, "nights_detail": detail,
		})
	}
}
