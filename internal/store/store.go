// Package store provides an in-memory data store for the PoC.
// In the real Meshery implementation, this is replaced by the
// existing database layer with encrypted-at-rest credential storage.
package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/meshery/ai-adapter-poc/internal/models"
)

// Store is a thread-safe in-memory store for connections and credentials.
type Store struct {
	mu          sync.RWMutex
	connections map[string]*models.Connection
	credentials map[string]*models.Credential
}

// New creates an empty Store.
func New() *Store {
	return &Store{
		connections: make(map[string]*models.Connection),
		credentials: make(map[string]*models.Credential),
	}
}

// --- Credential Operations ---

func (s *Store) CreateCredential(cred *models.Credential) (*models.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cred.ID = uuid.New().String()
	cred.CreatedAt = time.Now()
	cred.UpdatedAt = time.Now()
	s.credentials[cred.ID] = cred
	return cred, nil
}

func (s *Store) GetCredential(id string) (*models.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cred, ok := s.credentials[id]
	if !ok {
		return nil, fmt.Errorf("credential not found: %s", id)
	}
	return cred, nil
}

func (s *Store) ListCredentials(userID string) []*models.Credential {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Credential
	for _, c := range s.credentials {
		if userID == "" || c.UserID == userID {
			result = append(result, c)
		}
	}
	return result
}

func (s *Store) DeleteCredential(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.credentials[id]; !ok {
		return fmt.Errorf("credential not found: %s", id)
	}
	delete(s.credentials, id)
	return nil
}

// --- Connection Operations ---

func (s *Store) CreateConnection(conn *models.Connection) (*models.Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn.ID = uuid.New().String()
	conn.CreatedAt = time.Now()
	conn.UpdatedAt = time.Now()
	if conn.Status == "" {
		conn.Status = models.StatusRegistered
	}
	s.connections[conn.ID] = conn
	return conn, nil
}

func (s *Store) GetConnection(id string) (*models.Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conn, ok := s.connections[id]
	if !ok {
		return nil, fmt.Errorf("connection not found: %s", id)
	}
	return conn, nil
}

func (s *Store) ListConnections(userID string) []*models.Connection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Connection
	for _, c := range s.connections {
		if userID == "" || c.UserID == userID {
			result = append(result, c)
		}
	}
	return result
}

func (s *Store) UpdateConnectionStatus(id string, status models.ConnectionStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, ok := s.connections[id]
	if !ok {
		return fmt.Errorf("connection not found: %s", id)
	}
	conn.Status = status
	conn.UpdatedAt = time.Now()
	return nil
}

func (s *Store) DeleteConnection(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.connections[id]; !ok {
		return fmt.Errorf("connection not found: %s", id)
	}
	delete(s.connections, id)
	return nil
}
