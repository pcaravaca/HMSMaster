package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/gorm"
)

var validChannels = map[string]bool{"airbnb": true, "booking": true, "expedia": true, "direct": true}

// channelSecretName resolves the per-channel HMAC secret env var. An empty
// secret means webhook signatures are not enforced (local development).
func channelSecretName(channel string) string {
	return "HMS_CHANNEL_" + strings.ToUpper(channel) + "_SECRET"
}

func verifyChannelSignature(channel string, body []byte, header string) bool {
	secret := os.Getenv(channelSecretName(channel))
	if secret == "" {
		return true
	}
	if !strings.HasPrefix(header, "sha256=") {
		return false
	}
	got, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return subtle.ConstantTimeCompare(got, mac.Sum(nil)) == 1
}

func channelConflicts(db *gorm.DB, roomID uint, checkIn, checkOut time.Time) (uint, bool) {
	var existing []models.Reservation
	db.Where("room_id = ? AND status NOT IN ?", roomID, []string{"canceled", "checked_out"}).Find(&existing)
	for _, r := range existing {
		if datesOverlap(checkIn, checkOut, r.CheckIn, r.CheckOut) {
			return r.ID, true
		}
	}
	return 0, false
}

type channelBookingInput struct {
	Channel    string `json:"channel"`
	ExternalID string `json:"external_id"`
	HotelID    uint   `json:"hotel_id"`
	RoomID     uint   `json:"room_id"`
	GuestName  string `json:"guest_name"`
	GuestEmail string `json:"guest_email"`
	CheckIn    string `json:"check_in"`
	CheckOut   string `json:"check_out"`
}

// createChannelBooking is the inbound OTA webhook. It is idempotent on
// (channel, external_id): a repeated delivery returns the existing booking
// instead of creating a second reservation.
func createChannelBooking(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
			return
		}
		var in channelBookingInput
		if err := json.Unmarshal(raw, &in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		in.Channel = strings.ToLower(strings.TrimSpace(in.Channel))
		if !validChannels[in.Channel] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported channel"})
			return
		}
		if !verifyChannelSignature(in.Channel, raw, c.GetHeader("X-HMS-Signature")) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
			return
		}
		if in.ExternalID == "" || in.HotelID == 0 || in.RoomID == 0 || in.GuestName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "external_id, hotel_id, room_id and guest_name are required"})
			return
		}
		checkIn, err1 := parseTimestamp(in.CheckIn)
		checkOut, err2 := parseTimestamp(in.CheckOut)
		if err1 != nil || err2 != nil || !checkOut.After(checkIn) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "valid check_in/check_out are required"})
			return
		}
		var existing models.ChannelBooking
		if err := db.Where("channel = ? AND external_id = ?", in.Channel, in.ExternalID).First(&existing).Error; err == nil {
			c.JSON(http.StatusOK, gin.H{"duplicate": true, "booking": existing, "reservation_id": existing.ReservationID})
			return
		}
		var room models.Room
		if err := db.First(&room, in.RoomID).Error; err != nil || room.HotelID != in.HotelID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "room not found in this hotel"})
			return
		}
		if room.Status == "maintenance" {
			c.JSON(http.StatusConflict, gin.H{"error": "room is under maintenance"})
			return
		}
		if id, conflict := channelConflicts(db, room.ID, checkIn, checkOut); conflict {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("room already reserved for those dates (reservation #%d)", id)})
			return
		}
		detail, total, err := quoteStay(db, in.HotelID, room.ID, checkIn, checkOut, room.BaseRate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rateJSON, _ := json.Marshal(detail)
		r := models.Reservation{
			HotelID: in.HotelID, RoomID: room.ID, RoomNumber: room.Number,
			GuestName: in.GuestName, GuestEmail: in.GuestEmail,
			CheckIn: checkIn, CheckOut: checkOut, TotalAmount: total,
			RateDetail: string(rateJSON), Status: "confirmed", PaymentStatus: "unpaid", Channel: in.Channel,
		}
		if err := db.Create(&r).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		booking := models.ChannelBooking{
			HotelID: in.HotelID, Channel: in.Channel, ExternalID: in.ExternalID,
			ReservationID: r.ID, GuestName: in.GuestName, CheckIn: checkIn, CheckOut: checkOut,
			Status: "confirmed", Payload: string(raw),
		}
		if err := db.Create(&booking).Error; err != nil {
			// Roll back the reservation if the dedup index rejects this booking.
			db.Delete(&r)
			c.JSON(http.StatusConflict, gin.H{"error": "duplicate external_id"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"booking": booking, "reservation": r})
	}
}

func listChannelBookings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := db.Model(&models.ChannelBooking{}).Order("id DESC")
		if v := c.Query("channel"); v != "" {
			q = q.Where("channel = ?", strings.ToLower(v))
		}
		if v := parseID(c.Query("hotel_id")); v != 0 {
			q = q.Where("hotel_id = ?", v)
		}
		var bookings []models.ChannelBooking
		if err := q.Find(&bookings).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, bookings)
	}
}

func cancelChannelBooking(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var b models.ChannelBooking
		if err := db.First(&b, parseID(c.Param("id"))).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
			return
		}
		if b.Status == "canceled" {
			c.JSON(http.StatusOK, gin.H{"message": "already canceled", "booking": b})
			return
		}
		b.Status = "canceled"
		db.Save(&b)
		db.Model(&models.Reservation{}).Where("id = ?", b.ReservationID).Update("status", "canceled")
		c.JSON(http.StatusOK, gin.H{"message": "booking canceled", "booking": b})
	}
}
