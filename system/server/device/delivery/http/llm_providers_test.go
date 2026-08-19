package http

import (
	"testing"
	"time"
)

func TestGrokTokRecordNeedsRefresh(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	fresh := grokTokRecord{ExpiresAt: now.Add(6 * time.Hour).Unix()}
	if fresh.needsRefresh(now) {
		t.Fatal("fresh token should not refresh")
	}
	soon := grokTokRecord{ExpiresAt: now.Add(time.Minute).Unix()}
	if !soon.needsRefresh(now) {
		t.Fatal("token inside skew should refresh")
	}
	legacy := grokTokRecord{
		SavedAt:   now.Add(-2 * time.Hour).Format(time.RFC3339),
		ExpiresIn: 3600,
	}
	if !legacy.needsRefresh(now) {
		t.Fatal("legacy saved_at+expires_in past skew should refresh")
	}
}
