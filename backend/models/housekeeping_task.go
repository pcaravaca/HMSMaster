package models

import (
	"time"

	"gorm.io/gorm"
)

// HousekeepingTask tracks room cleaning after check-out or on request.
type HousekeepingTask struct {
	gorm.Model
	HotelID       uint       `json:"hotel_id" gorm:"not null;index"`
	RoomID        uint       `json:"room_id" gorm:"not null;index"`
	RoomNumber    string     `json:"room_number"`
	ReservationID uint       `json:"reservation_id" gorm:"index"`
	Type          string     `json:"type" gorm:"default:'checkout_clean';size:30"`
	Status        string     `json:"status" gorm:"default:'pending';size:20"`
	AssignedTo    string     `json:"assigned_to" gorm:"size:80"`
	Notes         string     `json:"notes" gorm:"size:500"`
	CompletedAt   *time.Time `json:"completed_at"`
}
