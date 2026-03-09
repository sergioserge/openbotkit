package server

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/priyanshujain/openbotkit/internal/testutil"
	"github.com/priyanshujain/openbotkit/remote"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDocker_ServerE2E builds the Docker image from the repo's Dockerfile and
// tests the full remote deployment flow against the container.
// Requires Docker to be running.
func TestDocker_ServerE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Docker test in short mode")
	}

	// Check if Docker is available before attempting to use testcontainers
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("skipping Docker test: Docker is not running")
	}

	testutil.LoadEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	geminiKey := testutil.RequireGeminiKey(t)

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "../../",
			Dockerfile: "infrastructure/docker/Dockerfile",
		},
		ExposedPorts: []string{"8443/tcp"},
		Env: map[string]string{
			"OBK_CONFIG_DIR":    "/data",
			"OBK_AUTH_USERNAME": "testuser",
			"OBK_AUTH_PASSWORD": "testpass",
			"GEMINI_API_KEY":    geminiKey,
		},
		Cmd: []string{"server", "--addr", ":8443"},
		WaitingFor: wait.ForHTTP("/api/health").
			WithPort("8443/tcp").
			WithStartupTimeout(3 * time.Minute),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start container: %v", err)
	}
	defer container.Terminate(ctx)

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	port, err := container.MappedPort(ctx, "8443")
	if err != nil {
		t.Fatalf("get port: %v", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, port.Port())
	client := remote.NewClient(baseURL, "testuser", "testpass")

	// 1. Health check
	t.Run("health", func(t *testing.T) {
		health, err := client.Health()
		if err != nil {
			t.Fatalf("health: %v", err)
		}
		if health["status"] != "ok" {
			t.Fatalf("expected ok, got %q", health["status"])
		}
	})

	// 2. Memory CRUD
	t.Run("memory_crud", func(t *testing.T) {
		id, err := client.MemoryAdd("lives in Tokyo", "identity", "manual")
		if err != nil {
			t.Fatalf("add: %v", err)
		}
		if id == 0 {
			t.Fatal("expected non-zero ID")
		}

		items, err := client.MemoryList("")
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(items) == 0 {
			t.Fatal("expected at least 1 memory")
		}

		found := false
		for _, m := range items {
			if m.Content == "lives in Tokyo" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("added memory not found in list")
		}

		if err := client.MemoryDelete(id); err != nil {
			t.Fatalf("delete: %v", err)
		}
	})

	// 3. Auth enforcement
	t.Run("auth_required", func(t *testing.T) {
		noAuth := remote.NewClient(baseURL, "", "")

		// Health should work
		_, err := noAuth.Health()
		if err != nil {
			t.Fatalf("health without auth should work: %v", err)
		}

		// Memory should fail
		_, err = noAuth.MemoryList("")
		if err == nil {
			t.Fatal("expected auth error")
		}
	})

	// 4. Apple Notes push
	t.Run("applenotes_push", func(t *testing.T) {
		notes := []map[string]interface{}{
			{
				"apple_id":           "docker-note-1",
				"title":              "Docker Test Note",
				"body":               "This was pushed from a testcontainer",
				"folder":             "Test",
				"folder_id":          "f1",
				"account":            "iCloud",
				"password_protected": false,
				"created_at":         "2024-01-15T10:00:00Z",
				"modified_at":        "2024-01-15T10:00:00Z",
			},
		}
		if err := client.AppleNotesPush(notes); err != nil {
			t.Fatalf("push: %v", err)
		}
	})
}
