package store

import (
	"testing"

	"github.com/meshery/ai-adapter-poc/internal/models"
)

func TestStore_Credentials(t *testing.T) {
	s := New()

	cred := &models.Credential{
		Name:   "test-cred",
		Kind:   models.ProviderOpenAI,
		Secret: map[string]string{"key": "value"},
		UserID: "user1",
	}

	created, err := s.CreateCredential(cred)
	if err != nil {
		t.Fatalf("Failed to create credential: %v", err)
	}
	if created.ID == "" {
		t.Error("Expected credential ID to be populated")
	}

	fetched, err := s.GetCredential(created.ID)
	if err != nil {
		t.Fatalf("Failed to get credential: %v", err)
	}
	if fetched.Name != "test-cred" {
		t.Errorf("Expected name 'test-cred', got '%s'", fetched.Name)
	}

	creds := s.ListCredentials("user1")
	if len(creds) != 1 {
		t.Errorf("Expected 1 credential, got %d", len(creds))
	}

	err = s.DeleteCredential(created.ID)
	if err != nil {
		t.Fatalf("Failed to delete credential: %v", err)
	}

	_, err = s.GetCredential(created.ID)
	if err == nil {
		t.Error("Expected error fetching deleted credential")
	}
}

func TestStore_Connections(t *testing.T) {
	s := New()

	conn := &models.Connection{
		Name:   "test-conn",
		Kind:   models.ProviderOllama,
		UserID: "user1",
	}

	created, err := s.CreateConnection(conn)
	if err != nil {
		t.Fatalf("Failed to create connection: %v", err)
	}
	if created.ID == "" {
		t.Error("Expected connection ID to be populated")
	}
	if created.Status != models.StatusRegistered {
		t.Errorf("Expected status %s, got %s", models.StatusRegistered, created.Status)
	}

	fetched, err := s.GetConnection(created.ID)
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}
	if fetched.Name != "test-conn" {
		t.Errorf("Expected name 'test-conn', got '%s'", fetched.Name)
	}

	conns := s.ListConnections("user1")
	if len(conns) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(conns))
	}

	err = s.UpdateConnectionStatus(created.ID, models.StatusConnected)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	fetched, _ = s.GetConnection(created.ID)
	if fetched.Status != models.StatusConnected {
		t.Errorf("Expected status %s, got %s", models.StatusConnected, fetched.Status)
	}

	err = s.DeleteConnection(created.ID)
	if err != nil {
		t.Fatalf("Failed to delete connection: %v", err)
	}

	_, err = s.GetConnection(created.ID)
	if err == nil {
		t.Error("Expected error fetching deleted connection")
	}
}
