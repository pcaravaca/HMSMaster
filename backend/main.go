package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const version = "0.5.0"

func openDB() *gorm.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=*** dbname=hmsmaster port=5432 sslmode=disable"
	}
	var db *gorm.DB
	var err error
	for i := 1; i <= 15; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			return db
		}
		log.Printf("db not ready (attempt %d/15): %v", i, err)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("failed to connect database: %v", err)
	return nil
}

func main() {
	db := openDB()
	if err := db.AutoMigrate(&models.Hotel{}, &models.Room{}, &models.Reservation{}, &models.FolioCharge{}); err != nil {
		log.Fatalf("automigrate: %v", err)
	}
	seed(db)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "project": "HMSMaster", "version": version})
	})

	r.POST("/api/auth/login", loginHandler)

	api := r.Group("/api")
	api.Use(authMiddleware())
	{
		api.GET("/stats", statsHandler(db))
		api.GET("/availability", availabilityHandler(db))
		api.GET("/hotels", listHotels(db))
		api.POST("/hotels", createHotel(db))
		api.GET("/hotels/:id/rooms", listRooms(db))
		api.POST("/hotels/:id/rooms", createRoom(db))
		api.GET("/reservations", listReservations(db))
		api.POST("/reservations", createReservation(db))
		api.PATCH("/reservations/:id/status", updateReservationStatus(db))
		api.GET("/reservations/:id/charges", listCharges(db))
		api.POST("/reservations/:id/charges", addCharge(db))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("HMSMaster backend v%s listening on :%s", version, port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
