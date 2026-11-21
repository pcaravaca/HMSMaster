package main

import (
	"log"
	"os"
	"time"

	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/gorm"
)

// seed creates a demo property on first run so the dashboard is never empty.
func seed(db *gorm.DB) {
	if os.Getenv("HMS_SEED_DEMO") != "true" {
		log.Printf("demo seeding disabled (set HMS_SEED_DEMO=true to enable)")
		return
	}
	var count int64
	db.Model(&models.Hotel{}).Count(&count)
	if count > 0 {
		return
	}
	hotel := models.Hotel{
		Name:         "Hotel Demo Suite",
		Address:      "San José, Costa Rica",
		ContactEmail: "demo@hmsmaster.local",
		Phone:        "+506 2222-0000",
		Active:       true,
	}
	if err := db.Create(&hotel).Error; err != nil {
		log.Printf("seed hotel: %v", err)
		return
	}
	rooms := []models.Room{
		{HotelID: hotel.ID, Number: "101", Type: "standard", Floor: 1, BaseRate: 45000, Status: "available"},
		{HotelID: hotel.ID, Number: "102", Type: "standard", Floor: 1, BaseRate: 45000, Status: "available"},
		{HotelID: hotel.ID, Number: "103", Type: "standard", Floor: 1, BaseRate: 45000, Status: "occupied"},
		{HotelID: hotel.ID, Number: "104", Type: "standard", Floor: 1, BaseRate: 45000, Status: "cleaning"},
		{HotelID: hotel.ID, Number: "201", Type: "deluxe", Floor: 2, BaseRate: 70000, Status: "available"},
		{HotelID: hotel.ID, Number: "202", Type: "deluxe", Floor: 2, BaseRate: 70000, Status: "available"},
		{HotelID: hotel.ID, Number: "301", Type: "suite", Floor: 3, BaseRate: 120000, Status: "available"},
		{HotelID: hotel.ID, Number: "302", Type: "suite", Floor: 3, BaseRate: 120000, Status: "maintenance"},
	}
	db.Create(&rooms)

	guests := []models.Guest{
		{HotelID: hotel.ID, FirstName: "María", LastName: "Fernández", Email: "maria@example.com", Phone: "+506 8888-1111", DocumentType: "cedula", DocumentNumber: "1-2345-6789"},
		{HotelID: hotel.ID, FirstName: "John", LastName: "Carter", Email: "john@example.com", DocumentType: "pasaporte", DocumentNumber: "US1234567"},
	}
	db.Create(&guests)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, now.Location())
	res := []models.Reservation{
		{
			HotelID: hotel.ID, RoomID: rooms[2].ID, RoomNumber: "103",
			GuestID: guests[0].ID,
			GuestName: "María Fernández", GuestEmail: "maria@example.com",
			CheckIn: today.AddDate(0, 0, -1), CheckOut: today.AddDate(0, 0, 2),
			TotalAmount: 135000, Status: "checked_in", PaymentStatus: "partial", Channel: "direct",
		},
		{
			HotelID: hotel.ID, RoomID: rooms[4].ID, RoomNumber: "201",
			GuestID: guests[1].ID,
			GuestName: "John Carter", GuestEmail: "john@example.com",
			CheckIn: today, CheckOut: today.AddDate(0, 0, 3),
			TotalAmount: 210000, Status: "confirmed", PaymentStatus: "unpaid", Channel: "airbnb",
		},
	}
	db.Create(&res)
	db.Create(&models.FolioCharge{
		ReservationID: res[0].ID, Description: "Noche de habitación (103)",
		Amount: 45000, Source: "hms", SyncStatus: "local",
	})
	log.Printf("seeded demo hotel with %d rooms and %d reservations", len(rooms), len(res))
}
