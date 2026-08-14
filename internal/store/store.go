// Copyright 2026 The Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package store

import (
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/meshery/ai-adapter-poc/internal/models"
	"gorm.io/gorm"
)

// Store is a GORM-backed data store for connections and credentials.
type Store struct {
	db *gorm.DB
}

// New creates a new Store connected to an SQLite database.
func New() (*Store, error) {
	db, err := gorm.Open(sqlite.Open("meshery_ai_adapter.db"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// Auto Migrate the schemas
	err = db.AutoMigrate(&models.Connection{}, &models.Credential{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Store{db: db}, nil
}

// --- Credential Operations ---

func (s *Store) CreateCredential(cred *models.Credential) (*models.Credential, error) {
	cred.ID = uuid.New().String()
	cred.CreatedAt = time.Now()
	cred.UpdatedAt = time.Now()

	if err := s.db.Create(cred).Error; err != nil {
		return nil, err
	}
	return cred, nil
}

func (s *Store) GetCredential(id string) (*models.Credential, error) {
	var cred models.Credential
	if err := s.db.First(&cred, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("credential not found: %s", id)
	}
	return &cred, nil
}

func (s *Store) ListCredentials(userID string) []*models.Credential {
	var creds []*models.Credential
	query := s.db
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	query.Find(&creds)
	return creds
}

func (s *Store) DeleteCredential(id string) error {
	result := s.db.Delete(&models.Credential{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("credential not found: %s", id)
	}
	return nil
}

// --- Connection Operations ---

func (s *Store) CreateConnection(conn *models.Connection) (*models.Connection, error) {
	conn.ID = uuid.New().String()
	conn.CreatedAt = time.Now()
	conn.UpdatedAt = time.Now()
	if conn.Status == "" {
		conn.Status = models.StatusRegistered
	}

	if err := s.db.Create(conn).Error; err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *Store) GetConnection(id string) (*models.Connection, error) {
	var conn models.Connection
	if err := s.db.First(&conn, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("connection not found: %s", id)
	}
	return &conn, nil
}

func (s *Store) ListConnections(userID string) []*models.Connection {
	var conns []*models.Connection
	query := s.db
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	query.Find(&conns)
	return conns
}

func (s *Store) UpdateConnectionStatus(id string, status models.ConnectionStatus) error {
	result := s.db.Model(&models.Connection{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("connection not found: %s", id)
	}
	return nil
}

func (s *Store) DeleteConnection(id string) error {
	result := s.db.Delete(&models.Connection{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("connection not found: %s", id)
	}
	return nil
}
