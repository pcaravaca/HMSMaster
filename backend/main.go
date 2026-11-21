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

const version = "0.13.0"

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
	if err := db.AutoMigrate(&models.Hotel{}, &models.Room{}, &models.Guest{}, &models.Reservation{}, &models.FolioCharge{}, &models.RateRule{}, &models.User{}, &models.ChannelBooking{}, &models.HousekeepingTask{}); err != nil {
		log.Fatalf("automigrate: %v", err)
	}
	// Partial unique index on existing databases (empty keys are allowed).
	_ = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_folio_charges_idempotency_key ON folio_charges (idempotency_key) WHERE idempotency_key <> ''").Error
	_ = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_channel_bookings_external ON channel_bookings (channel, external_id)").Error
	seed(db)
	seedAdminUser(db)
	startPOSRetryWorker(db)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "project": "HMSMaster", "version": version})
	})

	r.POST("/api/auth/login", loginHandler(db))

	api := r.Group("/api")
	api.Use(authMiddleware())
	{
		api.GET("/auth/me", authMeHandler)
		api.GET("/reports/summary", reportSummaryHandler(db))
		api.GET("/housekeeping/tasks", listHousekeepingTasks(db))
		api.POST("/housekeeping/tasks", guardRoles(createHousekeepingTask(db), "admin","manager","reception","housekeeping"))
		api.PATCH("/housekeeping/tasks/:id", guardRoles(updateHousekeepingTask(db), "admin","manager","housekeeping"))
		api.POST("/channels/bookings", guardRoles(createChannelBooking(db), "admin","manager","reception"))
		api.GET("/channels/bookings", listChannelBookings(db))
		api.POST("/channels/bookings/:id/cancel", guardRoles(cancelChannelBooking(db), "admin","manager","reception"))
		api.GET("/stats", statsHandler(db))
		api.GET("/availability", availabilityHandler(db))
		api.GET("/rates", listRates(db))
		api.POST("/rates", guardRoles(createRate(db), "admin","manager"))
		api.DELETE("/rates/:id", guardRoles(deleteRate(db), "admin","manager"))
		api.GET("/rates/quote", quoteRate(db))
		api.GET("/posmaster/status", posStatusHandler(db))
		api.POST("/reservations/:id/charges/:chargeId/retry", guardRoles(retryChargeHandler(db), "admin","manager","accounting"))
		api.GET("/users", guardRoles(listUsers(db), "admin"))
		api.POST("/users", guardRoles(createUser(db), "admin"))
		api.PATCH("/users/:id", guardRoles(updateUser(db), "admin"))
		api.DELETE("/users/:id", guardRoles(deleteUser(db), "admin"))
		api.GET("/guests", listGuests(db))
		api.POST("/guests", guardRoles(createGuestHandler(db), "admin","manager","reception"))
		api.GET("/guests/:id", getGuest(db))
		api.PATCH("/guests/:id", guardRoles(updateGuest(db), "admin","manager","reception"))
		api.GET("/hotels", listHotels(db))
		api.POST("/hotels", guardRoles(createHotel(db), "admin","manager"))
		api.GET("/hotels/:id/rooms", listRooms(db))
		api.POST("/hotels/:id/rooms", guardRoles(createRoom(db), "admin","manager"))
		api.GET("/reservations", listReservations(db))
		api.POST("/reservations", guardRoles(createReservation(db), "admin","manager","reception"))
		api.PATCH("/reservations/:id/status", guardRoles(updateReservationStatus(db), "admin","manager","reception"))
		api.GET("/reservations/:id/charges", listCharges(db))
		api.POST("/reservations/:id/charges", guardRoles(addCharge(db), "admin","manager","accounting"))
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
