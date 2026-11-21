package models

import "gorm.io/gorm"

// RateRule overrides the base rate for a date range. RoomID == 0 means the
// rule applies to every room in the hotel; room-specific rules win.
type RateRule struct {
	gorm.Model
	HotelID   uint    `json:"hotel_id" gorm:"not null;index"`
	RoomID    uint    `json:"room_id" gorm:"index"`
	StartDate string  `json:"start_date" gorm:"size:10;not null;index"`
	EndDate   string  `json:"end_date" gorm:"size:10;not null;index"`
	Price     float64 `json:"price" gorm:"not null"`
	Notes     string  `json:"notes" gorm:"size:255"`
}
