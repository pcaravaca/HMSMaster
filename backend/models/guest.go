package models

import "gorm.io/gorm"

// Guest is the unified guest profile for a property. Reservations keep a
// snapshot of the name/email for historicalaccuracy, but link to this record
// so repeated stays share one profile and history.
type Guest struct {
	gorm.Model
	HotelID        uint   `json:"hotel_id" gorm:"not null;index"`
	FirstName      string `json:"first_name" gorm:"not null"`
	LastName       string `json:"last_name"`
	Email          string `json:"email" gorm:"index"`
	Phone          string `json:"phone"`
	DocumentType   string `json:"document_type"` // cedula, pasaporte, dimex...
	DocumentNumber string `json:"document_number" gorm:"index"`
	Notes          string `json:"notes"`
}

func (g Guest) FullName() string {
	if g.LastName == "" {
		return g.FirstName
	}
	return g.FirstName + " " + g.LastName
}
