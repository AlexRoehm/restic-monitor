package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentPoliciesRouterMapping tests that the route is registered
func TestAgentPoliciesRouterMapping(t *testing.T) {
	api, st := setupTestAPI(t)

	// Create test agent
	agent := store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	st.GetDB().Create(&agent)

	req := httptest.NewRequest("GET", "/agents/"+agent.ID.String()+"/policies", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	api.Handler().ServeHTTP(rec, req)

	// Should not be 404 (route exists)
	assert.NotEqual(t, http.StatusNotFound, rec.Code)
}

// TestGetAgentPolicies tests retrieving policies assigned to an agent
func TestGetAgentPolicies(t *testing.T) {
	api, st := setupTestAPI(t)

	// Create test agent
	agent := store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "backup-server",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	st.GetDB().Create(&agent)

	t.Run("Get policies for agent with no assignments", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/agents/"+agent.ID.String()+"/policies", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		policies, ok := response["policies"].([]interface{})
		require.True(t, ok)
		assert.Empty(t, policies)
	})

	t.Run("Get policies for agent with assignments", func(t *testing.T) {
		// Create test policies
		policy1 := store.Policy{
			TenantID: st.GetTenantID(),
			Name:     "daily-backup",
			Schedule: "0 2 * * *",
			IncludePaths: store.JSONB{
				"paths": []string{"/var/www", "/etc"},
			},
			RepositoryURL:  "s3:bucket1/path",
			RepositoryType: "s3",
			RepositoryConfig: store.JSONB{
				"type":   "s3",
				"bucket": "bucket1",
				"prefix": "path",
			},
			RetentionRules: store.JSONB{
				"keepDaily": 7,
			},
			Enabled: true,
		}
		st.GetDB().Create(&policy1)

		policy2 := store.Policy{
			TenantID:    st.GetTenantID(),
			Name:        "weekly-backup",
			Description: ptrString("Weekly full backup"),
			Schedule:    "0 3 * * 0",
			IncludePaths: store.JSONB{
				"paths": []string{"/home"},
			},
			ExcludePaths: store.JSONB{
				"patterns": []string{"*.log", "*.tmp"},
			},
			RepositoryURL:  "s3:bucket2/weekly",
			RepositoryType: "s3",
			RepositoryConfig: store.JSONB{
				"type":   "s3",
				"bucket": "bucket2",
				"prefix": "weekly",
			},
			RetentionRules: store.JSONB{
				"keepWeekly": 4,
			},
			BandwidthLimitKBps: ptrInt(5120),
			ParallelFiles:      ptrInt(8),
			Enabled:            true,
		}
		st.GetDB().Create(&policy2)

		// Assign policies to agent
		link1 := store.AgentPolicyLink{
			AgentID:  agent.ID,
			PolicyID: policy1.ID,
		}
		st.GetDB().Create(&link1)

		link2 := store.AgentPolicyLink{
			AgentID:  agent.ID,
			PolicyID: policy2.ID,
		}
		st.GetDB().Create(&link2)

		req := httptest.NewRequest("GET", "/agents/"+agent.ID.String()+"/policies", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		policies, ok := response["policies"].([]interface{})
		require.True(t, ok)
		assert.Len(t, policies, 2)

		// Verify first policy structure
		p1 := policies[0].(map[string]interface{})
		assert.Equal(t, "daily-backup", p1["name"])
		assert.Equal(t, "0 2 * * *", p1["schedule"])

		includePaths := p1["includePaths"].([]interface{})
		assert.Len(t, includePaths, 2)
		assert.Contains(t, includePaths, "/var/www")
		assert.Contains(t, includePaths, "/etc")

		repository := p1["repository"].(map[string]interface{})
		assert.Equal(t, "s3", repository["type"])
		assert.Equal(t, "bucket1", repository["bucket"])

		retentionRules := p1["retentionRules"].(map[string]interface{})
		assert.Equal(t, float64(7), retentionRules["keepDaily"])

		// Verify second policy has optional fields
		p2 := policies[1].(map[string]interface{})
		assert.Equal(t, "weekly-backup", p2["name"])
		assert.Equal(t, "Weekly full backup", p2["description"])
		assert.Equal(t, float64(5120), p2["bandwidthLimitKBps"])
		assert.Equal(t, float64(8), p2["parallelFiles"])

		excludePaths := p2["excludePaths"].([]interface{})
		assert.Len(t, excludePaths, 2)
	})

	t.Run("Agent not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest("GET", "/agents/"+nonExistentID.String()+"/policies", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Invalid agent ID format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/agents/invalid-uuid/policies", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Unauthorized access", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/agents/"+agent.ID.String()+"/policies", nil)
		// No Authorization header
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

// TestAgentPolicySerializationFormat tests the serialization format
func TestAgentPolicySerializationFormat(t *testing.T) {
	api, st := setupTestAPI(t)

	// Create test agent
	agent := store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "serialization-test",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	st.GetDB().Create(&agent)

	t.Run("Policy serialization excludes orchestrator metadata", func(t *testing.T) {
		// Create policy with all fields
		policy := store.Policy{
			TenantID:    st.GetTenantID(),
			Name:        "test-policy",
			Description: ptrString("Test description"),
			Schedule:    "0 2 * * *",
			IncludePaths: store.JSONB{
				"paths": []string{"/data"},
			},
			ExcludePaths: store.JSONB{
				"patterns": []string{"*.log"},
			},
			RepositoryURL:  "s3:test-bucket/prefix",
			RepositoryType: "s3",
			RepositoryConfig: store.JSONB{
				"type":   "s3",
				"bucket": "test-bucket",
				"prefix": "prefix",
				"region": "us-east-1",
			},
			RetentionRules: store.JSONB{
				"keepDaily":   7,
				"keepWeekly":  4,
				"keepMonthly": 12,
			},
			BandwidthLimitKBps: ptrInt(10240),
			ParallelFiles:      ptrInt(4),
			Enabled:            true,
		}
		st.GetDB().Create(&policy)

		// Assign to agent
		link := store.AgentPolicyLink{
			AgentID:  agent.ID,
			PolicyID: policy.ID,
		}
		st.GetDB().Create(&link)

		req := httptest.NewRequest("GET", "/agents/"+agent.ID.String()+"/policies", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		policies := response["policies"].([]interface{})
		require.Len(t, policies, 1)

		p := policies[0].(map[string]interface{})

		// Should NOT include orchestrator metadata
		assert.NotContains(t, p, "id")
		assert.NotContains(t, p, "tenantId")
		assert.NotContains(t, p, "enabled")
		assert.NotContains(t, p, "createdAt")
		assert.NotContains(t, p, "updatedAt")

		// Should include agent-relevant fields
		assert.Contains(t, p, "name")
		assert.Contains(t, p, "description")
		assert.Contains(t, p, "schedule")
		assert.Contains(t, p, "includePaths")
		assert.Contains(t, p, "excludePaths")
		assert.Contains(t, p, "repository")
		assert.Contains(t, p, "retentionRules")
		assert.Contains(t, p, "bandwidthLimitKBps")
		assert.Contains(t, p, "parallelFiles")

		// Verify data integrity
		assert.Equal(t, "test-policy", p["name"])
		assert.Equal(t, "Test description", p["description"])
		assert.Equal(t, "0 2 * * *", p["schedule"])
	})

	t.Run("Only enabled policies are returned", func(t *testing.T) {
		// Create a new agent for this test
		agent2 := store.Agent{
			TenantID: st.GetTenantID(),
			Hostname: "enabled-test-agent",
			OS:       "linux",
			Arch:     "amd64",
			Version:  "1.0.0",
			Status:   "online",
		}
		st.GetDB().Create(&agent2)

		// Create enabled and disabled policies
		enabledPolicy := store.Policy{
			TenantID: st.GetTenantID(),
			Name:     "enabled-policy",
			Schedule: "0 2 * * *",
			IncludePaths: store.JSONB{
				"paths": []string{"/data"},
			},
			RepositoryURL:  "s3:bucket/path",
			RepositoryType: "s3",
			RepositoryConfig: store.JSONB{
				"type":   "s3",
				"bucket": "bucket",
			},
			RetentionRules: store.JSONB{
				"keepDaily": 7,
			},
			Enabled: true,
		}
		st.GetDB().Create(&enabledPolicy)

		disabledPolicy := store.Policy{
			TenantID: st.GetTenantID(),
			Name:     "disabled-policy",
			Schedule: "0 3 * * *",
			IncludePaths: store.JSONB{
				"paths": []string{"/data"},
			},
			RepositoryURL:  "s3:bucket/path",
			RepositoryType: "s3",
			RepositoryConfig: store.JSONB{
				"type":   "s3",
				"bucket": "bucket",
			},
			RetentionRules: store.JSONB{
				"keepDaily": 7,
			},
			Enabled: true, // Create as enabled first
		}
		st.GetDB().Create(&disabledPolicy)
		// Then disable it
		st.GetDB().Model(&disabledPolicy).Update("enabled", false)

		// Assign both to agent
		st.GetDB().Create(&store.AgentPolicyLink{
			AgentID:  agent2.ID,
			PolicyID: enabledPolicy.ID,
		})
		st.GetDB().Create(&store.AgentPolicyLink{
			AgentID:  agent2.ID,
			PolicyID: disabledPolicy.ID,
		})

		req := httptest.NewRequest("GET", "/agents/"+agent2.ID.String()+"/policies", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		api.Handler().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		policies := response["policies"].([]interface{})
		assert.Len(t, policies, 1) // Only enabled policy

		p := policies[0].(map[string]interface{})
		assert.Equal(t, "enabled-policy", p["name"])
	})
}

// Helper functions
func ptrString(s string) *string {
	return &s
}

func ptrInt(i int) *int {
	return &i
}
