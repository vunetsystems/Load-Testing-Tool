package generator

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// SessionManager manages session ID rotation
type SessionManager struct {
	mu               sync.RWMutex
	currentSessionID string
	currentCommandID string
	prefix           string
	rotationMode     string        // "time" or "count"
	rotationInterval time.Duration // for time-based rotation
	rotationCount    int           // threshold for count-based rotation
	currentCount     int           // current message count
	lastRotation     time.Time
}

// NewSessionManager creates a new session manager
func NewSessionManager(prefix string, rotationMode string, rotationInterval time.Duration, rotationCount int) *SessionManager {
	sm := &SessionManager{
		prefix:           prefix,
		rotationMode:     rotationMode,
		rotationInterval: rotationInterval,
		rotationCount:    rotationCount,
		currentCount:     0,
	}
	sm.rotate()
	return sm
}

// GetCurrentSession returns the current session ID and command ID, rotating if necessary
func (sm *SessionManager) GetCurrentSession() (string, string) {
	sm.mu.RLock()
	needsRotation := sm.shouldRotate()
	if !needsRotation {
		sessionID := sm.currentSessionID
		commandID := sm.currentCommandID
		sm.mu.RUnlock()
		
		// Increment counter for count-based mode (need write lock)
		if sm.rotationMode == "count" {
			sm.mu.Lock()
			sm.currentCount++
			sm.mu.Unlock()
		}
		
		return sessionID, commandID
	}
	sm.mu.RUnlock()

	// Need to rotate
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Double-check after acquiring write lock
	if sm.shouldRotate() {
		sm.rotate()
	}
	
	// Increment counter for count-based mode
	if sm.rotationMode == "count" {
		sm.currentCount++
	}

	return sm.currentSessionID, sm.currentCommandID
}

// shouldRotate checks if rotation is needed based on mode (must be called with at least read lock)
func (sm *SessionManager) shouldRotate() bool {
	if sm.rotationMode == "count" {
		return sm.currentCount >= sm.rotationCount
	}
	// Default to time-based rotation
	return time.Since(sm.lastRotation) >= sm.rotationInterval
}

// rotate generates a new session ID (must be called with write lock held)
func (sm *SessionManager) rotate() {
	epochMillis := time.Now().UnixMilli()
	randomSuffix := generateRandomAlphanumeric(10)
	sm.currentSessionID = fmt.Sprintf("%d%s%s", epochMillis, sm.prefix, randomSuffix)
	
	// Select random command ID
	if len(AllCommandIDs) > 0 {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(AllCommandIDs))))
		sm.currentCommandID = AllCommandIDs[n.Int64()]
	}
	
	// Reset counter for count-based rotation
	sm.currentCount = 0
	sm.lastRotation = time.Now()
}

// GetRotationCount returns how many times the session has rotated
func (sm *SessionManager) GetRotationCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	elapsed := time.Since(sm.lastRotation)
	return int(elapsed / sm.rotationInterval)
}
