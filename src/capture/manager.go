// src/capture/manager.go
package capture

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	instance *Manager
	once     sync.Once
)

type Manager struct {
	mu         sync.RWMutex
	sessions   map[string]*CaptureSession
	outputPath string
}

type CaptureSession struct {
	ID         string    `json:"id"`
	Domain     string    `json:"domain"`
	Protocol   string    `json:"protocol"` // "tls", "quic", or "both"
	MaxPackets int       `json:"max_packets"`
	Count      int       `json:"count"`
	StartTime  time.Time `json:"start_time"`
	Active     bool      `json:"active"`
	Captures   []Capture `json:"captures"`
	mu         sync.Mutex
}

type Capture struct {
	Protocol  string    `json:"protocol"`
	Domain    string    `json:"domain"`
	Timestamp time.Time `json:"timestamp"`
	Size      int       `json:"size"`
	Filepath  string    `json:"filepath"`
	HexData   string    `json:"hex_data,omitempty"`
}

func GetManager() *Manager {
	once.Do(func() {
		instance = &Manager{
			sessions:   make(map[string]*CaptureSession),
			outputPath: "./captures",
		}
		os.MkdirAll(instance.outputPath, 0755)
	})
	return instance
}

func (m *Manager) StartSession(domain, protocol string, maxPackets int) (*CaptureSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionID := fmt.Sprintf("cap_%d", time.Now().Unix())

	session := &CaptureSession{
		ID:         sessionID,
		Domain:     domain,
		Protocol:   protocol,
		MaxPackets: maxPackets,
		StartTime:  time.Now(),
		Active:     true,
		Captures:   []Capture{},
	}

	m.sessions[sessionID] = session
	return session, nil
}

func (m *Manager) GetSession(id string) (*CaptureSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

func (m *Manager) ListSessions() []*CaptureSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*CaptureSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

func (m *Manager) StopSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}

	session.Active = false
	return nil
}

func (m *Manager) CapturePayload(domain, protocol string, payload []byte) bool {
	m.mu.RLock()
	activeSessions := []*CaptureSession{}
	for _, session := range m.sessions {
		if session.Active {
			activeSessions = append(activeSessions, session)
		}
	}
	m.mu.RUnlock()

	captured := false
	for _, session := range activeSessions {
		if session.shouldCapture(domain, protocol) {
			if err := m.saveCapture(session, protocol, domain, payload); err == nil {
				captured = true
			}
		}
	}

	return captured
}

func (s *CaptureSession) shouldCapture(domain, protocol string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.Active || s.Count >= s.MaxPackets {
		return false
	}

	// Check domain match (support wildcards)
	if s.Domain != "*" && s.Domain != domain {
		return false
	}

	// Check protocol match
	if s.Protocol != "both" && s.Protocol != protocol {
		return false
	}

	return true
}

func (m *Manager) saveCapture(session *CaptureSession, protocol, domain string, payload []byte) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.Count >= session.MaxPackets {
		return fmt.Errorf("max packets reached")
	}

	timestamp := time.Now()
	filename := fmt.Sprintf("%s_%s_%s_%d.bin",
		session.ID, protocol, sanitizeDomain(domain), timestamp.Unix())
	filepath := filepath.Join(m.outputPath, filename)

	// Save binary
	if err := os.WriteFile(filepath, payload, 0644); err != nil {
		return err
	}

	capture := Capture{
		Protocol:  protocol,
		Domain:    domain,
		Timestamp: timestamp,
		Size:      len(payload),
		Filepath:  filepath,
		HexData:   hex.EncodeToString(payload),
	}

	session.Captures = append(session.Captures, capture)
	session.Count++

	if session.Count >= session.MaxPackets {
		session.Active = false
	}

	return nil
}

func sanitizeDomain(domain string) string {
	// Replace dots and special chars for filename
	result := ""
	for _, ch := range domain {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' {
			result += string(ch)
		} else if ch == '.' {
			result += "_"
		}
	}
	return result
}
