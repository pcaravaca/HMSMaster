package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/gorm"
)

type posChargeResult struct {
	OK         bool
	Ref        string
	Err        string
	Definitive bool // true when POSMaster rejected the request (safe to retry)
}

func posmasterConfigured() bool {
	return os.Getenv("POSMASTER_API_URL") != ""
}

func idempotencyKey(chargeID, reservationID uint) string {
	return fmt.Sprintf("hms-c%d-r%d", chargeID, reservationID)
}

func observationForKey(key string) string {
	return "HMSMaster · " + key
}

// syncChargeToPOSMaster forwards a folio charge to POSMaster's invoicing API.
// The idempotency key travels in observaciones so an existing invoice can be
// found later without creating a duplicate.
func syncChargeToPOSMaster(description string, amount float64, reservationID, chargeID uint) posChargeResult {
	base := os.Getenv("POSMASTER_API_URL")
	if base == "" {
		return posChargeResult{Err: "POSMASTER_API_URL not configured"}
	}
	formaPago := os.Getenv("POSMASTER_FORMA_PAGO")
	if formaPago == "" {
		formaPago = "efectivo"
	}
	key := idempotencyKey(chargeID, reservationID)
	payload := map[string]any{
		"forma_pago":    formaPago,
		"total_pagado":  amount,
		"observaciones": observationForKey(key),
		"detalles": []map[string]any{
			{
				"cantidad": 1, "descripcion": description,
				"precio_unitario": amount, "porcentaje_iva": 13.0, "descuento": 0,
			},
		},
	}
	if tiendaID := os.Getenv("POSMASTER_TIENDA_ID"); tiendaID != "" {
		payload["tienda_id"] = tiendaID
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", strings.TrimRight(base, "/")+"/api/facturas", bytes.NewReader(body))
	if err != nil {
		return posChargeResult{Err: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	if tok := os.Getenv("POSMASTER_API_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return posChargeResult{Err: err.Error()} // outcome unknown: never auto-retry
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		definitive := resp.StatusCode < 500 && resp.StatusCode != 408 && resp.StatusCode != 429
		return posChargeResult{
			Err: fmt.Sprintf("posmaster responded %d", resp.StatusCode),
			Definitive: definitive,
		}
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return posChargeResult{OK: true}
	}
	ref := ""
	if id, ok := out["id"]; ok {
		ref = fmt.Sprintf("%v", id)
	} else if f, ok := out["factura"].(map[string]any); ok {
		if id, ok := f["id"]; ok {
			ref = fmt.Sprintf("%v", id)
		}
	}
	return posChargeResult{OK: true, Ref: ref}
}

// findPOSInvoiceByKey looks for an invoice created by a previous attempt with
// the same key. It limits the search to the last 7 days and the configured
// store, matching POSMaster's GetFacturas filters.
func findPOSInvoiceByKey(key string, since time.Time) (string, error) {
	base := os.Getenv("POSMASTER_API_URL")
	if base == "" {
		return "", fmt.Errorf("bridge not configured")
	}
	client := &http.Client{Timeout: 8 * time.Second}
	// Search around the charge's own date instead of a fixed 7-day window, and
	// walk every page so a busy store cannot hide the invoice behind page 1.
	day := since
	if day.IsZero() {
		day = time.Now()
	}
	const pageSize = 200
	const maxPages = 20
	seen := 0
	for page := 1; page <= maxPages; page++ {
		url := fmt.Sprintf("%s/api/facturas?limit=%d&page=%d&desde=%s&hasta=%s",
			strings.TrimRight(base, "/"), pageSize, page,
			day.AddDate(0, 0, -1).Format("2006-01-02"), day.AddDate(0, 0, 2).Format("2006-01-02"))
		if tiendaID := os.Getenv("POSMASTER_TIENDA_ID"); tiendaID != "" {
			url += "&tienda_id=" + tiendaID
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", err
		}
		if tok := os.Getenv("POSMASTER_API_TOKEN"); tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return "", fmt.Errorf("posmaster list responded %d", resp.StatusCode)
		}
		var list struct {
			Facturas []struct {
				ID            string  `json:"id"`
				Observaciones *string `json:"observaciones"`
			} `json:"facturas"`
			Paginacion struct {
				Total int `json:"total"`
			} `json:"paginacion"`
		}
		err = json.NewDecoder(resp.Body).Decode(&list)
		resp.Body.Close()
		if err != nil {
			return "", err
		}
		for _, f := range list.Facturas {
			if f.Observaciones != nil && strings.Contains(*f.Observaciones, key) {
				return f.ID, nil
			}
		}
		seen += len(list.Facturas)
		if list.Paginacion.Total > 0 && seen >= list.Paginacion.Total {
			break
		}
		if len(list.Facturas) < pageSize {
			break
		}
	}
	return "", nil
}

// retryPOSCharge is idempotent: it first looks for an existing invoice, then
// only posts again when the previous failure was definitive (or force=true).
func retryPOSCharge(db *gorm.DB, ch *models.FolioCharge, force bool) string {
	if !posmasterConfigured() {
		return "bridge not configured"
	}
	if ch.SyncStatus == "synced" {
		return "already synced"
	}
	if ch.IdempotencyKey == "" {
		ch.IdempotencyKey = idempotencyKey(ch.ID, ch.ReservationID)
	}
	if !force && ch.SyncAttempts >= 3 {
		return "attempt limit reached (use force=true)"
	}
	if ref, err := findPOSInvoiceByKey(ch.IdempotencyKey, ch.CreatedAt); err == nil && ref != "" {
		ch.SyncStatus = "synced"
		ch.POSRef = ref
		ch.SyncError = ""
		db.Save(ch)
		return "recovered existing invoice"
	}
	if ch.SyncStatus == "uncertain" && !force {
		return "outcome unknown: use force=true after manual reconciliation"
	}
	var r models.Reservation
	if err := db.First(&r, ch.ReservationID).Error; err != nil {
		return "reservation not found"
	}
	ch.SyncAttempts++
	res := syncChargeToPOSMaster(ch.Description, ch.Amount, r.ID, ch.ID)
	if res.OK {
		ch.SyncStatus = "synced"
		ch.POSRef = res.Ref
		ch.SyncError = ""
	} else if res.Definitive {
		ch.SyncStatus = "failed"
		ch.SyncError = res.Err
	} else {
		ch.SyncStatus = "uncertain"
		ch.SyncError = res.Err
	}
	db.Save(ch)
	return "retry attempted: " + res.Err
}

func startPOSRetryWorker(db *gorm.DB) {
	if !posmasterConfigured() {
		return
	}
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			var charges []models.FolioCharge
			db.Where("sync_status = ? AND sync_attempts < ?", "failed", 3).Find(&charges)
			for i := range charges {
				retryPOSCharge(db, &charges[i], false)
			}
		}
	}()
}

func retryChargeHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ch models.FolioCharge
		if err := db.First(&ch, parseID(c.Param("chargeId"))).Error; err != nil || ch.ReservationID != parseID(c.Param("id")) {
			c.JSON(http.StatusNotFound, gin.H{"error": "charge not found"})
			return
		}
		force := c.Query("force") == "true"
		msg := retryPOSCharge(db, &ch, force)
		c.JSON(http.StatusOK, gin.H{"message": msg, "charge": ch})
	}
}

func posStatusHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var local, synced, failed, uncertain int64
		db.Model(&models.FolioCharge{}).Where("sync_status = ?", "local").Count(&local)
		db.Model(&models.FolioCharge{}).Where("sync_status = ?", "synced").Count(&synced)
		db.Model(&models.FolioCharge{}).Where("sync_status = ?", "failed").Count(&failed)
		db.Model(&models.FolioCharge{}).Where("sync_status = ?", "uncertain").Count(&uncertain)
		c.JSON(http.StatusOK, gin.H{
			"configured": posmasterConfigured(), "local": local, "synced": synced,
			"failed": failed, "uncertain": uncertain,
		})
	}
}
