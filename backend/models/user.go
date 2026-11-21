package models

import "gorm.io/gorm"

// User is a local HMSMaster account. Passwords are stored as bcrypt hashes.
type User struct {
	gorm.Model
	Username     string `json:"username" gorm:"uniqueIndex;not null;size:80"`
	PasswordHash string `json:"-" gorm:"not null;size:100"`
	Role         string `json:"role" gorm:"not null;default:'reception';size:20"`
	HotelID      uint   `json:"hotel_id" gorm:"index"`
	Active       bool   `json:"active" gorm:"default:true"`
}
