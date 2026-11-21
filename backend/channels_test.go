package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func signBody(secret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

func TestChannelSignature(t *testing.T) {
	body := []byte(`{"external_id":"ABC-1"}`)
	t.Setenv(channelSecretName("airbnb"), "")
	if !verifyChannelSignature("airbnb", body, "") {
		t.Fatal("missing secret should allow local development")
	}
	secret := "webhook-test-secret"
	t.Setenv(channelSecretName("airbnb"), secret)
	if channelSecretName("booking") != "HMS_CHANNEL_BOOKING_SECRET" {
		t.Fatalf("wrong env name: %s", channelSecretName("booking"))
	}
	if !verifyChannelSignature("airbnb", body, signBody(secret, body)) {
		t.Fatal("valid signature rejected")
	}
	for _, bad := range []string{"", "sha256=deadbeef", "md5=abc", "sha256=zz", signBody("other", body)} {
		if verifyChannelSignature("airbnb", body, bad) {
			t.Fatalf("invalid signature accepted: %q", bad)
		}
	}
	if !verifyChannelSignature("booking", body, "") {
		t.Fatal("other channel secret should not affect booking")
	}
}
