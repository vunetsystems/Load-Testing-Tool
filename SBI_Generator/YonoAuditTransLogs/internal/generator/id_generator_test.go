package generator

import (
	"testing"
	"time"
)

func TestGenerateTransactionID(t *testing.T) {
	pattern := "cRhKTOYInnV"
	
	// Generate multiple IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateTransactionID(pattern)
		
		// Check uniqueness
		if ids[id] {
			t.Errorf("Duplicate transaction ID generated: %s", id)
		}
		ids[id] = true
		
		// Check format (should contain pattern)
		if len(id) < len(pattern) {
			t.Errorf("Transaction ID too short: %s", id)
		}
	}
}

func TestGenerateRequestNo(t *testing.T) {
	length := 27
	
	reqNo := GenerateRequestNo(length)
	
	if len(reqNo) != length {
		t.Errorf("Expected length %d, got %d", length, len(reqNo))
	}
	
	// Check all digits
	for _, c := range reqNo {
		if c < '0' || c > '9' {
			t.Errorf("Non-digit character in request number: %c", c)
		}
	}
}

func TestGenerateTraceID(t *testing.T) {
	tests := []struct {
		name   string
		format string
		length int
	}{
		{"UUID format", "uuid", 0},
		{"Hex format 32", "hex", 32},
		{"Hex format 16", "hex", 16},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traceID := GenerateTraceID(tt.format, tt.length)
			
			if tt.format == "uuid" {
				// UUID should be 36 characters (with hyphens)
				if len(traceID) != 36 {
					t.Errorf("UUID length should be 36, got %d", len(traceID))
				}
			} else {
				if len(traceID) != tt.length {
					t.Errorf("Expected length %d, got %d", tt.length, len(traceID))
				}
			}
		})
	}
}

func TestSessionManager(t *testing.T) {
	prefix := "TEST"
	interval := 100 * time.Millisecond
	
	sm := NewSessionManager(prefix, interval)
	
	// Get initial session
	session1, cmd1 := sm.GetCurrentSession()
	if session1 == "" {
		t.Error("Session ID should not be empty")
	}
	if cmd1 == "" && len(AllCommandIDs) > 0 {
		t.Error("Command ID should not be empty if AllCommandIDs is populated")
	}
	
	// Get session again immediately (should be same)
	session2, cmd2 := sm.GetCurrentSession()
	if session1 != session2 {
		t.Error("Session ID should not change before rotation interval")
	}
	if cmd1 != cmd2 {
		t.Error("Command ID should not change before rotation interval")
	}
	
	// Wait for rotation
	time.Sleep(150 * time.Millisecond)
	
	// Get session after rotation
	session3, _ := sm.GetCurrentSession()
	if session1 == session3 {
		t.Error("Session ID should change after rotation interval")
	}
}
