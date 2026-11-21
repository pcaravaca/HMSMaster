package models

import (
	"time"

	"gorm.io/gorm"
)

type Hotel struct {
	gorm.Model
	Name         string `json:"name" gorm:"not null"`
	Address      string `json:"address" gorm:"not null"`
	ContactEmail string `json:"contact_email"`
	Phone        string `json:"phone"`
	Active       bool   `json:"active" gorm:"default:true"`
}

type Room struct {
	gorm.Model
	HotelID  uint    `json:"hotel_id" gorm:"not null;index"`
	Number   string  `json:"number" gorm:"not null"`
	Type     string  `json:"type" gorm:"default:'standard'"` // standard, deluxe, suite
	Floor    int     `json:"floor"`
	BaseRate float64 `json:"base_rate" gorm:"not null"`
	Status   string  `json:"status" gorm:"default:'available'"` // available, occupied, cleaning, maintenance
}

type Reservation struct {
	gorm.Model
	HotelID       uint      `json:"hotel_id" gorm:"not null;index"`
	RoomID        uint      `json:"room_id" gorm:"index"`
	GuestName     string    `json:"guest_name" gorm:"not null"`
	GuestEmail    string    `json:"guest_email"`
	CheckIn       time.Time `json:"check_in" gorm:"not null"`
	CheckOut      time.Time `json:"check_out" gorm:"not null"`
	RoomNumber    string    `json:"room_number" gorm:"index"`
	TotalAmount   float64   `json:"total_amount" gorm:"not null"`
	Status        string    `json:"status" gorm:"default:'pending'"`        // pending, confirmed, checked_in, checked_out, canceled
	PaymentStatus string    `json:"payment_status" gorm:"default:'unpaid'"` // unpaid, paid, partial
	Channel       string    `json:"channel" gorm:"default:'direct'"`        // direct, airbnb, booking, expedia
}

// FolioCharge is a charge on a guest folio. Charges can originate in HMS
// (room night, minibar, service) or be synced to POSMaster for invoicing.
type FolioCharge struct {
	gorm.Model
	ReservationID uint    `json:"reservation_id" gorm:"not null;index"`
	Description   string  `json:"description" gorm:"not null"`
	Amount        float64 `json:"amount" gorm:"not null"`
	Source        string  `json:"source" gorm:"default:'hms'"` // hms, posmaster
	POSRef        string  `json:"pos_ref"`                     // POSMaster sale id once synced
	SyncStatus    string  `json:"sync_status" gorm:"default:'local'"` // local, synced, failed
}
