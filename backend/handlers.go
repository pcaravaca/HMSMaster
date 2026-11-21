package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/gorm"
)

func statsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var hotels, rooms, reservations, occupied, inHouse, arrivals int64
		roomScope := func() *gorm.DB { return db.Model(&models.Room{}) }
		resScope := func() *gorm.DB { return db.Model(&models.Reservation{}) }
		if hid := c.Query("hotel_id"); hid != "" {
			id := parseID(hid)
			roomScope = func() *gorm.DB { return db.Model(&models.Room{}).Where("hotel_id = ?", id) }
			resScope = func() *gorm.DB { return db.Model(&models.Reservation{}).Where("hotel_id = ?", id) }
			db.Model(&models.Hotel{}).Where("id = ?", id).Count(&hotels)
		} else {
			db.Model(&models.Hotel{}).Count(&hotels)
		}
		roomScope().Count(&rooms)
		resScope().Count(&reservations)
		roomScope().Where("status = ?", "occupied").Count(&occupied)
		resScope().Where("status = ?", "checked_in").Count(&inHouse)
		today := time.Now().Format("2006-01-02")
		resScope().Where("DATE(check_in) = ? AND status IN ?", today, []string{"pending", "confirmed"}).Count(&arrivals)
		occupancy := 0.0
		if rooms > 0 {
			occupancy = float64(occupied) / float64(rooms) * 100
		}
		c.JSON(http.StatusOK, gin.H{
			"hotels": hotels, "rooms": rooms, "reservations": reservations,
			"occupied_rooms": occupied, "in_house": inHouse,
			"arrivals_today": arrivals, "occupancy_pct": occupancy,
		})
	}
}

func listHotels(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var hotels []models.Hotel
		db.Order("id").Find(&hotels)
		c.JSON(http.StatusOK, hotels)
	}
}

func createHotel(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var h models.Hotel
		if err := c.ShouldBindJSON(&h); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if h.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		if err := db.Create(&h).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, h)
	}
}

func listRooms(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rooms []models.Room
		db.Where("hotel_id = ?", c.Param("id")).Order("number").Find(&rooms)
		c.JSON(http.StatusOK, rooms)
	}
}

func createRoom(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var room models.Room
		if err := c.ShouldBindJSON(&room); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		room.HotelID = parseID(c.Param("id"))
		if room.Number == "" || room.BaseRate <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "number and base_rate (>0) are required"})
			return
		}
		if err := db.Create(&room).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, room)
	}
}

func listReservations(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var res []models.Reservation
		q := db.Order("check_in")
		if s := c.Query("status"); s != "" {
			q = q.Where("status = ?", s)
		}
		if hid := c.Query("hotel_id"); hid != "" {
			q = q.Where("hotel_id = ?", parseID(hid))
		}
		q.Find(&res)
		c.JSON(http.StatusOK, res)
	}
}

func createReservation(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r models.Reservation
		if err := c.ShouldBindJSON(&r); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if r.HotelID == 0 || r.CheckIn.IsZero() || r.CheckOut.IsZero() || (r.GuestID == 0 && r.GuestName == "") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hotel_id, check_in, check_out and guest_name (or guest_id) are required"})
			return
		}
		if !r.CheckOut.After(r.CheckIn) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "check_out must be after check_in"})
			return
		}
		if r.RoomID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "room_id is required"})
			return
		}
		if r.RoomID != 0 {
			var room models.Room
			if err := db.First(&room, r.RoomID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "room not found"})
				return
			}
			r.RoomNumber = room.Number
			r.HotelID = room.HotelID

			// A room cannot be sold twice for overlapping nights. Only active
			// reservations count; canceled and checked_out bookings are history.
			var existing []models.Reservation
			db.Where("room_id = ? AND status NOT IN ?", r.RoomID, []string{"canceled", "checked_out"}).Find(&existing)
			for _, ex := range existing {
				if datesOverlap(r.CheckIn, r.CheckOut, ex.CheckIn, ex.CheckOut) {
					c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("room %s already reserved for those dates (reservation #%d)", room.Number, ex.ID)})
					return
				}
			}
			if room.Status == "maintenance" {
				c.JSON(http.StatusConflict, gin.H{"error": "room is under maintenance"})
				return
			}

			// Server-side pricing: date/range rules override the base rate.
			// A caller may still pass a contracted total (e.g. channel manager).
			if r.TotalAmount <= 0 {
				detail, total, err := quoteStay(db, r.HotelID, r.RoomID, r.CheckIn, r.CheckOut, room.BaseRate)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				r.TotalAmount = total
				if raw, err := json.Marshal(detail); err == nil {
					r.RateDetail = string(raw)
				}
			}
		}
		if r.GuestID != 0 {
			var guest models.Guest
			if err := db.First(&guest, r.GuestID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "guest not found"})
				return
			}
			if guest.HotelID != r.HotelID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "guest belongs to a different hotel"})
				return
			}
			if r.GuestName == "" {
				r.GuestName = guest.FullName()
			}
			if r.GuestEmail == "" {
				r.GuestEmail = guest.Email
			}
		}
		if r.Status == "" {
			r.Status = "pending"
		}
		if err := db.Create(&r).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, r)
	}
}

var allowedTransitions = map[string][]string{
	"pending":     {"confirmed", "canceled"},
	"confirmed":   {"checked_in", "canceled"},
	"checked_in":  {"checked_out"},
	"checked_out": {},
	"canceled":    {},
}

func updateReservationStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Status string `json:"status"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var r models.Reservation
		if err := db.First(&r, parseID(c.Param("id"))).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "reservation not found"})
			return
		}
		ok := false
		for _, s := range allowedTransitions[r.Status] {
			if s == body.Status {
				ok = true
				break
			}
		}
		if !ok {
			c.JSON(http.StatusConflict, gin.H{"error": "invalid transition " + r.Status + " -> " + body.Status})
			return
		}
		r.Status = body.Status
		db.Save(&r)
		if r.RoomID != 0 {
			switch body.Status {
			case "checked_in":
				db.Model(&models.Room{}).Where("id = ?", r.RoomID).Update("status", "occupied")
			case "checked_out":
				db.Model(&models.Room{}).Where("id = ?", r.RoomID).Update("status", "cleaning")
				ensureCheckoutTask(db, &r)
			case "canceled":
				db.Model(&models.Room{}).Where("id = ?", r.RoomID).Update("status", "available")
			}
		}
		c.JSON(http.StatusOK, r)
	}
}

func listCharges(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var charges []models.FolioCharge
		db.Where("reservation_id = ?", c.Param("id")).Order("id").Find(&charges)
		c.JSON(http.StatusOK, charges)
	}
}

func addCharge(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ch models.FolioCharge
		if err := c.ShouldBindJSON(&ch); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var r models.Reservation
		if err := db.First(&r, parseID(c.Param("id"))).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "reservation not found"})
			return
		}
		if ch.Description == "" || ch.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "description and amount (>0) are required"})
			return
		}
		ch.ReservationID = r.ID
		ch.Source = "hms"
		ch.SyncStatus = "local"
		if err := db.Create(&ch).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if ch.IdempotencyKey == "" {
			ch.IdempotencyKey = idempotencyKey(ch.ID, r.ID)
			db.Save(&ch)
		}
		// Suite integration: the charge is persisted first so its idempotency
		// key is stable across retries. Uncertain outcomes are never retried
		// automatically to avoid duplicate invoices.
		if posmasterConfigured() {
			log.Printf("charge %d POSMaster sync: %s", ch.ID, retryPOSCharge(db, &ch, false))
		}
		c.JSON(http.StatusCreated, ch)
	}
}

func parseID(s string) uint {
	var id uint
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		id = id*10 + uint(ch-'0')
	}
	return id
}
