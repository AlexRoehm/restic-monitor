package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPolicyRoutes tests that policy routes are registered (TDD)
func TestPolicyRoutes(t *testing.T) {
	api, _ := setupTestAPI(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"POST /policies", "POST", "/policies"},
		{"GET /policies", "GET", "/policies"},
		{"GET /policies/{id}", "GET", "/policies/550e8400-e29b-41d4-a716-446655440000"},
		{"PUT /policies/{id}", "PUT", "/policies/550e8400-e29b-41d4-a716-446655440000"},
		{"DELETE /policies/{id}", "DELETE", "/policies/550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.method == "POST" || tt.method == "PUT" {
				policy := map[string]interface{}{
					"name":         "test-policy",
					"schedule":     "0 2 * * *",
					"includePaths": []string{"/data"},
					"repository": map[string]interface{}{
						"type":   "s3",
						"bucket": "my-bucket",
					},
					"retentionRules": map[string]interface{}{
						"keepDaily": 7,
					},
				}
				body, _ = json.Marshal(policy)
			}

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			api.Handler().ServeHTTP(rec, req)

			// Routes not implemented yet (TDD) - should get 404
			// After implementation, these will return appropriate status codes
			t.Logf("Response status: %d", rec.Code)
		})
	}
}

// TestCreatePolicy tests POST /policies endpoint (TDD)
func TestCreatePolicy(t *testing.T) {
	api, st := setupTestAPI(t)

	t.Run("Create policy with minimal fields", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":         "daily-backup",
			"schedule":     "0 2 * * *",
			"includePaths": []string{"/home", "/var/www"},
			"repository": map[string]interface{}{
				"type":   "s3",
				"bucket": "my-backups",
				"prefix": "server1",
				"region": "us-west-2",
			},
			"retentionRules": map[string]interface{}{
				"keepDaily":   7,
				"keepWeekly":  4,
				"keepMonthly": 12,
			},
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/policies", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		assert.NotEmpty(t, response["id"])
		assert.Equal(t, "daily-backup", response["name"])
		assert.Equal(t, "0 2 * * *", response["schedule"])
		assert.NotNil(t, response["createdAt"])
		assert.NotNil(t, response["updatedAt"])
		_ = st
	})

	t.Run("Create policy with optional fields", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":         "limited-backup",
			"description":  "Backup with bandwidth limits",
			"schedule":     "0 3 * * *",
			"includePaths": []string{"/data"},
			"excludePaths": []string{"*.tmp", "*.log"},
			"repository": map[string]interface{}{
				"type": "rest-server",
				"url":  "https://backup.example.com:8000/repo",
			},
			"retentionRules": map[string]interface{}{
				"keepLast":  10,
				"keepDaily": 14,
			},
			"bandwidthLimitKBps": 5120,
			"parallelFiles":      2,
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/policies", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "limited-backup", response["name"])
		assert.Equal(t, "Backup with bandwidth limits", response["description"])
		assert.Equal(t, float64(5120), response["bandwidthLimitKBps"])
		assert.Equal(t, float64(2), response["parallelFiles"])
	})

	t.Run("Missing required fields", func(t *testing.T) {
		payload := map[string]interface{}{
			"name": "incomplete-policy",
			// Missing schedule, includePaths, repository, retentionRules
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/policies", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Duplicate policy name", func(t *testing.T) {
		// Create first policy
		payload1 := map[string]interface{}{
			"name":         "unique-name-test",
			"schedule":     "0 2 * * *",
			"includePaths": []string{"/data"},
			"repository": map[string]interface{}{
				"type":   "s3",
				"bucket": "bucket1",
			},
			"retentionRules": map[string]interface{}{
				"keepDaily": 7,
			},
		}

		body1, _ := json.Marshal(payload1)
		req1 := httptest.NewRequest("POST", "/policies", bytes.NewReader(body1))
		req1.Header.Set("Authorization", "Bearer test-token")
		req1.Header.Set("Content-Type", "application/json")
		rec1 := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec1, req1)
		assert.Equal(t, http.StatusCreated, rec1.Code)

		// Try to create second policy with same name
		payload2 := map[string]interface{}{
			"name":         "unique-name-test",
			"schedule":     "0 3 * * *",
			"includePaths": []string{"/other"},
			"repository": map[string]interface{}{
				"type":   "s3",
				"bucket": "bucket2",
			},
			"retentionRules": map[string]interface{}{
				"keepDaily": 7,
			},
		}

		body2, _ := json.Marshal(payload2)
		req2 := httptest.NewRequest("POST", "/policies", bytes.NewReader(body2))
		req2.Header.Set("Authorization", "Bearer test-token")
		req2.Header.Set("Content-Type", "application/json")
		rec2 := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec2, req2)
		assert.Equal(t, http.StatusConflict, rec2.Code)
	})

	t.Run("Unauthorized access", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":     "test-policy",
			"schedule": "0 2 * * *",
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/policies", bytes.NewReader(body))
		// No Authorization header
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

// TestListPolicies tests GET /policies endpoint (TDD)
func TestListPolicies(t *testing.T) {
	api, st := setupTestAPI(t)

	// Create some test policies
	tenantID := st.GetTenantID()

	policy1 := store.Policy{
		TenantID:       tenantID,
		Name:           "policy-1",
		Schedule:       "0 2 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3:bucket/path1",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepDaily": 7},
		Enabled:        true,
	}
	st.GetDB().Create(&policy1)

	policy2 := store.Policy{
		TenantID:       tenantID,
		Name:           "policy-2",
		Schedule:       "0 3 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/var"}},
		RepositoryURL:  "s3:bucket/path2",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepDaily": 14},
		Enabled:        false,
	}
	st.GetDB().Create(&policy2)

	t.Run("List all policies", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/policies", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response []map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(response), 2)

		// Verify policy fields are present
		for _, p := range response {
			assert.NotEmpty(t, p["id"])
			assert.NotEmpty(t, p["name"])
			assert.NotEmpty(t, p["schedule"])
			assert.NotNil(t, p["enabled"])
		}
	})

	t.Run("Empty list when no policies", func(t *testing.T) {
		// Clean up policies
		st.GetDB().Where("1=1").Delete(&store.Policy{})

		req := httptest.NewRequest("GET", "/policies", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response []map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, 0, len(response))
	})
}

// TestGetPolicy tests GET /policies/{id} endpoint (TDD)
func TestGetPolicy(t *testing.T) {
	api, st := setupTestAPI(t)

	// Create a test policy
	tenantID := st.GetTenantID()

	description := "Test policy description"
	policy := store.Policy{
		TenantID:           tenantID,
		Name:               "test-policy",
		Description:        &description,
		Schedule:           "0 2 * * *",
		IncludePaths:       store.JSONB{"paths": []string{"/home", "/var"}},
		ExcludePaths:       store.JSONB{"patterns": []string{"*.tmp"}},
		RepositoryURL:      "s3:bucket/test",
		RepositoryType:     "s3",
		RepositoryConfig:   store.JSONB{"bucket": "test", "region": "us-west-2"},
		RetentionRules:     store.JSONB{"keepDaily": 7, "keepWeekly": 4},
		BandwidthLimitKBps: intPtr(10240),
		ParallelFiles:      intPtr(4),
		Enabled:            true,
	}
	st.GetDB().Create(&policy)

	t.Run("Get existing policy", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/policies/"+policy.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, policy.ID.String(), response["id"])
		assert.Equal(t, "test-policy", response["name"])
		assert.Equal(t, "Test policy description", response["description"])
		assert.Equal(t, "0 2 * * *", response["schedule"])
		assert.NotNil(t, response["includePaths"])
		assert.NotNil(t, response["excludePaths"])
		assert.NotNil(t, response["repository"])
		assert.NotNil(t, response["retentionRules"])
		assert.Equal(t, float64(10240), response["bandwidthLimitKBps"])
		assert.Equal(t, float64(4), response["parallelFiles"])
		assert.Equal(t, true, response["enabled"])
	})

	t.Run("Policy not found", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		req := httptest.NewRequest("GET", "/policies/"+nonExistentID, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Invalid UUID format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/policies/invalid-uuid", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// TestUpdatePolicy tests PUT /policies/{id} endpoint (TDD)
func TestUpdatePolicy(t *testing.T) {
	api, st := setupTestAPI(t)

	// Create a test policy
	tenantID := st.GetTenantID()

	policy := store.Policy{
		TenantID:       tenantID,
		Name:           "original-name",
		Schedule:       "0 2 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3:bucket/original",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepDaily": 7},
		Enabled:        true,
	}
	st.GetDB().Create(&policy)

	t.Run("Update policy fields", func(t *testing.T) {
		update := map[string]interface{}{
			"name":         "updated-name",
			"description":  "Updated description",
			"schedule":     "0 3 * * *",
			"includePaths": []string{"/data", "/home"},
			"retentionRules": map[string]interface{}{
				"keepDaily":  14,
				"keepWeekly": 8,
			},
			"bandwidthLimitKBps": 5120,
		}

		body, _ := json.Marshal(update)
		req := httptest.NewRequest("PUT", "/policies/"+policy.ID.String(), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "updated-name", response["name"])
		assert.Equal(t, "Updated description", response["description"])
		assert.Equal(t, "0 3 * * *", response["schedule"])
		assert.Equal(t, float64(5120), response["bandwidthLimitKBps"])
		assert.NotEqual(t, response["createdAt"], response["updatedAt"])
	})

	t.Run("Policy not found", func(t *testing.T) {
		update := map[string]interface{}{
			"name": "new-name",
		}

		nonExistentID := uuid.New().String()
		body, _ := json.Marshal(update)
		req := httptest.NewRequest("PUT", "/policies/"+nonExistentID, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Cannot update ID", func(t *testing.T) {
		newID := uuid.New().String()
		update := map[string]interface{}{
			"id":   newID,
			"name": "should-not-update-id",
		}

		body, _ := json.Marshal(update)
		req := httptest.NewRequest("PUT", "/policies/"+policy.ID.String(), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		// ID should remain unchanged
		var response map[string]interface{}
		json.NewDecoder(rec.Body).Decode(&response)
		assert.Equal(t, policy.ID.String(), response["id"])
		assert.NotEqual(t, newID, response["id"])
	})
}

// TestDeletePolicy tests DELETE /policies/{id} endpoint (TDD)
func TestDeletePolicy(t *testing.T) {
	api, st := setupTestAPI(t)

	tenantID := st.GetTenantID()

	t.Run("Delete existing policy", func(t *testing.T) {
		policy := store.Policy{
			TenantID:       tenantID,
			Name:           "to-delete",
			Schedule:       "0 2 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/data"}},
			RepositoryURL:  "s3:bucket/test",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keepDaily": 7},
			Enabled:        true,
		}
		st.GetDB().Create(&policy)

		req := httptest.NewRequest("DELETE", "/policies/"+policy.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)

		// Verify policy is deleted
		var count int64
		st.GetDB().Model(&store.Policy{}).Where("id = ?", policy.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Policy not found", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		req := httptest.NewRequest("DELETE", "/policies/"+nonExistentID, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Cannot delete policy with agent assignments", func(t *testing.T) {
		// Create policy and agent
		policy := store.Policy{
			TenantID:       tenantID,
			Name:           "assigned-policy",
			Schedule:       "0 2 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/data"}},
			RepositoryURL:  "s3:bucket/test",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keepDaily": 7},
			Enabled:        true,
		}
		st.GetDB().Create(&policy)

		agent := createTestAgent(t, st, "assigned-policy-test")

		// Create agent-policy link
		link := store.AgentPolicyLink{
			AgentID:  agent.ID,
			PolicyID: policy.ID,
		}
		st.GetDB().Create(&link)

		req := httptest.NewRequest("DELETE", "/policies/"+policy.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)

		// Verify policy still exists
		var count int64
		st.GetDB().Model(&store.Policy{}).Where("id = ?", policy.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

// Helper function for int pointer
func intPtr(i int) *int {
	return &i
}
