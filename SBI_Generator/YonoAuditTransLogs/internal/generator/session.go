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
	rotationInterval time.Duration
	lastRotation     time.Time
}

// NewSessionManager creates a new session manager
func NewSessionManager(prefix string, rotationInterval time.Duration) *SessionManager {
	sm := &SessionManager{
		prefix:           prefix,
		rotationInterval: rotationInterval,
	}
	sm.rotate()
	return sm
}

// GetCurrentSession returns the current session ID and command ID, rotating if necessary
func (sm *SessionManager) GetCurrentSession() (string, string) {
	sm.mu.RLock()
	if time.Since(sm.lastRotation) < sm.rotationInterval {
		sessionID := sm.currentSessionID
		commandID := sm.currentCommandID
		sm.mu.RUnlock()
		return sessionID, commandID
	}
	sm.mu.RUnlock()

	// Need to rotate
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(sm.lastRotation) >= sm.rotationInterval {
		sm.rotate()
	}

	return sm.currentSessionID, sm.currentCommandID
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
	
	sm.lastRotation = time.Now()
}

// GetRotationCount returns how many times the session has rotated
func (sm *SessionManager) GetRotationCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	elapsed := time.Since(sm.lastRotation)
	return int(elapsed / sm.rotationInterval)
}
