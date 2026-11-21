package models

import (
	"time"

	"gorm.io/gorm"
)

// ChannelBooking is an incoming OTA booking, deduplicated by external id.
type ChannelBooking struct {
	gorm.Model
	HotelID       uint      `json:"hotel_id" gorm:"not null;index"`
	Channel       string    `json:"channel" gorm:"not null;size:20;index"`
	ExternalID    string    `json:"external_id" gorm:"not null;size:120"`
	ReservationID uint      `json:"reservation_id" gorm:"index"`
	GuestName     string    `json:"guest_name"`
	CheckIn       time.Time `json:"check_in"`
	CheckOut      time.Time `json:"check_out"`
	Status        string    `json:"status" gorm:"default:'received';size:20"`
	Payload       string    `json:"payload" gorm:"type:text"`
}
