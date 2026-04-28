package generator

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type SessionManager struct {
	currentSessionID string
	mu               sync.RWMutex
}

func NewSessionManager() *SessionManager {
	sm := &SessionManager{}
	sm.rotateSession()
	go sm.startRotation()
	return sm
}

func (sm *SessionManager) GetCurrentSession() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentSessionID
}

func (sm *SessionManager) startRotation() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		sm.rotateSession()
	}
}

func (sm *SessionManager) rotateSession() {
	// Generate new session ID using timestamp
	timestamp := time.Now().UnixNano()
	newSessionID := fmt.Sprintf("%d", timestamp)

	sm.mu.Lock()
	sm.currentSessionID = newSessionID
	sm.mu.Unlock()

	log.Printf("Regenerating session ID: %s", newSessionID)
}
