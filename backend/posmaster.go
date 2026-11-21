package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type posChargeResult struct {
	OK  bool
	Ref string
	Err string
}

func posmasterConfigured() bool {
	return os.Getenv("POSMASTER_API_URL") != ""
}

// syncChargeToPOSMaster forwards a folio charge to POSMaster's invoicing API
// (POST /api/facturas, free-text line). Contract verified against
// POSMaster backend/internal/handlers/facturas.go (facturaRequestInput):
// detalles[] accepts descripcion + precio_unitario + porcentaje_iva -
// WITHOUT a catalog product. Requires POSMASTER_API_URL, POSMASTER_API_TOKEN
// (admin/cajero JWT) and POSMASTER_TIENDA_ID; without them charges stay local.
func syncChargeToPOSMaster(description string, amount float64, reservationID uint) posChargeResult {
	base := os.Getenv("POSMASTER_API_URL")
	if base == "" {
		return posChargeResult{Err: "POSMASTER_API_URL not configured"}
	}
	formaPago := os.Getenv("POSMASTER_FORMA_PAGO")
	if formaPago == "" {
		formaPago = "efectivo"
	}
	iva := 13.0
	obs := fmt.Sprintf("HMSMaster · Reserva #%d", reservationID)
	payload := map[string]any{
		"forma_pago":    formaPago,
		"total_pagado":  amount,
		"observaciones": obs,
		"detalles": []map[string]any{
			{
				"cantidad":        1,
				"descripcion":     description,
				"precio_unitario": amount,
				"porcentaje_iva":  iva,
				"descuento":       0,
			},
		},
	}
	if tiendaID := os.Getenv("POSMASTER_TIENDA_ID"); tiendaID != "" {
		payload["tienda_id"] = tiendaID
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", base+"/api/facturas", bytes.NewReader(body))
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
		return posChargeResult{Err: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return posChargeResult{Err: fmt.Sprintf("posmaster responded %d", resp.StatusCode)}
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
