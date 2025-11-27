# EPIC 16 — Unified Multi-Agent Backup Policy Orchestration & User-Friendly Configuration Framework

**Status:** 📋 Planning  
**Priority:** High  
**Date Started:** 2025-11-26  
**Target Completion:** TBD

---

## Overview

EPIC 16 is a **super-epic** that transforms the orchestrator into a central control plane with comprehensive policy-driven orchestration capabilities. This epic consolidates and extends multiple planned features:

- **Original EPIC 16**: Multi-repo + multi-agent orchestration
- **New Filesystem Browser**: Visual path selection on agents
- **Enhanced Rule Builder**: Include/exclude pattern designer
- **Agent Sandbox**: Security enforcement for path access
- **Credential Management**: Secure storage and delivery (moved from Epic 10)
- **Hook Integration**: Pre/post backup hooks (coordination with Epic 14)

### Strategic Importance

This epic forms the **foundation for all advanced backup orchestration features** (Epics 17-21):
- Epic 17 (Preflight Validation) depends on policy validation framework
- Epic 18 (Smart Triggering) depends on policy engine
- Epic 19 (Forecasting) depends on policy analytics
- Epic 20 (DR + Restore) depends on credential management
- Epic 20B (Policy Wizard) depends on visual rule builder

---

## Current State Assessment

### ✅ Already Implemented (EPICs 2-15)

| Feature | Status | Source | Notes |
|---------|--------|--------|-------|
| **Policy CRUD** | ✅ Complete | EPIC 6 | Full API + validation + 147 tests |
| **Policy-Agent Assignment** | ✅ Complete | EPIC 7 | Many-to-many links + 16 tests |
| **Basic Policy UI** | ✅ Complete | Phase 1-2 | Create/Edit/Delete/Assign/Detach modals |
| **Repository Config** | ✅ Basic | Policy model | URL + type + config JSONB |
| **Scheduler Integration** | ✅ Complete | EPIC 14 | Cron + interval scheduling |
| **Backup Execution** | ✅ Complete | EPIC 11 | Agent executes policies |
| **Concurrency Control** | ✅ Complete | EPIC 15 | Per-agent task limits |

### ⚠️ Partially Implemented

| Feature | Current State | Gap | Action Needed |
|---------|--------------|-----|---------------|
| **Include/Exclude Paths** | Simple array in Policy | No visual builder, no validation | Build UI tree + pattern preview |
| **Repository Types** | Type field exists | No credential store, no cert handling | Add Credential model + encryption |
| **Retention Rules** | JSONB in Policy | No conflict detection | Add policy validation engine |
| **Agent Metadata** | JSONB field exists | No sandbox definition | Add AgentSandbox model |

### ❌ Not Yet Implemented

| Feature | Required By User Story | Complexity | Priority |
|---------|----------------------|------------|----------|
| **Agent Filesystem Browser API** | US #3 | High | P0 |
| **Visual Include/Exclude Builder** | US #4 | Medium | P0 |
| **Agent Sandbox Engine** | US #5 | High | P0 |
| **Credential Store** | US #6 | High | P0 |
| **Hook Integration Layer** | US #7 | Medium | P1 |
| **Policy Conflict Detection** | US #1 | Medium | P0 |
| **Policy-Agent Link UI** | US #8 | Low | P1 |

---

## User Story Breakdown

### User Story 1 — Backup Policy CRUD System (Unified)

**Goal**: Create, edit, and manage backup policies centrally with conflict detection

**Status**: ⚠️ 70% Complete

**What's Done**:
- ✅ Policy model with all required fields (internal/store/models.go)
- ✅ CRUD API endpoints (internal/api/policy.go)
- ✅ Schema validation (147 tests in internal/api/policy_test.go)
- ✅ Basic UI for create/edit/delete (frontend/src/App.vue)
- ✅ Repository type field + config JSONB
- ✅ Retention rules JSONB
- ✅ Include/exclude paths arrays

**What's Missing**:
- ❌ Sandbox configuration in Policy model
- ❌ Credentials ID reference in Policy model
- ❌ Hook references (preHooks, postHooks arrays)
- ❌ Conflict detection:
  - Includes referencing forbidden sandbox paths
  - Hooks requiring access to forbidden paths
  - Invalid schedule validation
  - Retention rule conflicts

**Implementation Plan**:

1. **Phase 1A: Extend Policy Model** (4h)
   ```go
   type Policy struct {
       // Existing fields...
       
       // EPIC 16 additions
       SandboxConfig    JSONB     `gorm:"serializer:json" json:"sandbox_config,omitempty"`
       CredentialsID    *uuid.UUID `gorm:"type:uuid" json:"credentials_id,omitempty"`
       PreHooks         JSONB     `gorm:"serializer:json" json:"pre_hooks,omitempty"`  // Array of hook IDs
       PostHooks        JSONB     `gorm:"serializer:json" json:"post_hooks,omitempty"` // Array of hook IDs
   }
   
   type SandboxConfig struct {
       Allowed   []string `json:"allowed"`
       Forbidden []string `json:"forbidden"`
       MaxDepth  int      `json:"max_depth"`
   }
   ```
   
   **Tests (TDD-first)**:
   - Policy with sandbox config validation
   - Credentials ID foreign key constraint
   - Hook ID array validation
   - JSON schema validation for SandboxConfig

2. **Phase 1B: Conflict Detection Service** (6h)
   ```go
   type PolicyValidator struct {
       db *gorm.DB
   }
   
   func (v *PolicyValidator) ValidatePolicy(policy *Policy) (*ValidationResult, error) {
       // Check includes vs sandbox
       // Check hooks vs sandbox
       // Check schedule syntax
       // Check retention rules
       // Check credentials exist
   }
   ```
   
   **Tests (TDD-first)**:
   - Include path in forbidden directory → error
   - Hook accessing forbidden path → error
   - Invalid cron expression → error
   - Missing credentials ID → error
   - Conflicting retention rules → warning

3. **Phase 1C: Migration** (2h)
   - Migration 007: Add sandbox_config, credentials_id, pre_hooks, post_hooks columns
   - Migrate existing policies to new schema (NULL sandbox = unrestricted)

**Estimated Effort**: 12 hours  
**Tests**: +45 new tests

---

### User Story 2 — Agent-Side Policy Application

**Goal**: Agents receive and validate policies with all orchestration details

**Status**: ⚠️ 60% Complete

**What's Done**:
- ✅ Agent polls for tasks (EPIC 9)
- ✅ Policy data included in task payload
- ✅ Repository settings in task
- ✅ Include/exclude patterns in task
- ✅ Retention rules in task

**What's Missing**:
- ❌ Credentials not delivered to agent
- ❌ Hook definitions not in task payload
- ❌ Agent-side sandbox validation
- ❌ Policy version tracking
- ❌ Preflight validation (Epic 17 dependency)

**Implementation Plan**:

1. **Phase 2A: Extend Task Model** (3h)
   ```go
   type Task struct {
       // Existing fields...
       
       // EPIC 16 additions
       CredentialsToken  *string   `gorm:"type:text" json:"credentials_token,omitempty"`  // Time-limited
       PreHooks          JSONB     `gorm:"serializer:json" json:"pre_hooks,omitempty"`
       PostHooks         JSONB     `gorm:"serializer:json" json:"post_hooks,omitempty"`
       SandboxConfig     JSONB     `gorm:"serializer:json" json:"sandbox_config,omitempty"`
       PolicyVersion     *int      `json:"policy_version,omitempty"`
   }
   ```

2. **Phase 2B: Agent-Side Validation** (4h)
   - Agent receives task with sandbox config
   - Agent validates paths against sandbox before execution
   - Agent validates hook commands against sandbox
   - Agent caches policy version locally
   
   **Location**: `agent/validator.go`
   
   ```go
   type TaskValidator struct {
       sandboxEngine *SandboxEngine
   }
   
   func (v *TaskValidator) ValidateTask(task *Task) error {
       // Validate paths against sandbox
       // Validate hooks against sandbox
       // Verify credentials token not expired
   }
   ```

3. **Phase 2C: Policy Versioning** (2h)
   - Add version number to Policy model
   - Increment on every update
   - Agent compares versions, refetches if changed

**Estimated Effort**: 9 hours  
**Tests**: +30 new tests

---

### User Story 3 — Agent Filesystem Browser API

**Goal**: Expose agent's filesystem tree via API for visual path selection

**Status**: ❌ Not Started

**Dependencies**:
- Agent Sandbox Engine (US #5) - must enforce sandbox on browsing

**Implementation Plan**:

1. **Phase 3A: Agent Filesystem Service** (6h)
   
   **New Package**: `agent/pkg/filesystem/`
   
   ```go
   type FilesystemService struct {
       sandbox *SandboxEngine
   }
   
   // GET /agent/fs/root
   func (s *FilesystemService) GetRoots() ([]FSNode, error)
   
   // GET /agent/fs/tree?path=/var
   func (s *FilesystemService) GetTree(path string, depth int, showHidden bool) ([]FSNode, error)
   
   type FSNode struct {
       Path       string    `json:"path"`
       Name       string    `json:"name"`
       Type       string    `json:"type"`  // file, dir, symlink
       Size       int64     `json:"size"`
       ModTime    time.Time `json:"mod_time"`
       Children   []FSNode  `json:"children,omitempty"`
       Accessible bool      `json:"accessible"` // False if sandbox blocks
   }
   ```
   
   **Features**:
   - Only shows paths allowed by sandbox
   - Pagination for large directories (>10,000 entries)
   - Hidden file toggle
   - Symlink detection
   - Permission checking

2. **Phase 3B: Orchestrator Proxy** (3h)
   
   **New Endpoint**: `GET /api/v1/agents/{id}/filesystem/tree`
   
   Orchestrator proxies request to agent, caches response for 24h
   
   ```go
   func (h *AgentHandler) GetFilesystemTree(c *gin.Context) {
       agentID := c.Param("id")
       path := c.Query("path")
       
       // Check cache
       if cached := h.cache.Get(agentID, path); cached != nil {
           c.JSON(200, cached)
           return
       }
       
       // Forward to agent
       tree, err := h.agentClient.GetFilesystemTree(agentID, path)
       if err != nil {
           c.JSON(500, gin.H{"error": err.Error()})
           return
       }
       
       // Cache for 24h
       h.cache.Set(agentID, path, tree, 24*time.Hour)
       
       c.JSON(200, tree)
   }
   ```

3. **Phase 3C: Large Directory Handling** (2h)
   - Implement cursor-based pagination
   - Stream results for very large dirs
   - Timeout protection (30s max)

**Tests (TDD-first)**:
- Sandbox-allowed paths accessible
- Sandbox-forbidden paths hidden
- Large directory pagination (>10k files)
- Hidden files toggle
- Symlink handling
- Depth enforcement
- Timeout handling
- Cache expiry

**Estimated Effort**: 11 hours  
**Tests**: +35 new tests

---

### User Story 4 — Visual Include/Exclude Rule Builder UI

**Goal**: UI for visually selecting paths and generating Restic patterns

**Status**: ❌ Not Started

**Dependencies**:
- Agent Filesystem Browser API (US #3)

**Implementation Plan**:

1. **Phase 4A: Tree Component** (8h)
   
   **New Vue Component**: `frontend/src/components/FilesystemTree.vue`
   
   ```vue
   <template>
     <div class="filesystem-tree">
       <div class="tree-toolbar">
         <input type="search" v-model="searchQuery" placeholder="Search paths..." />
         <label><input type="checkbox" v-model="showHidden" /> Show hidden</label>
       </div>
       
       <div class="tree-view">
         <FSNode
           v-for="node in rootNodes"
           :key="node.path"
           :node="node"
           :selected="selectedPaths"
           :excluded="excludedPaths"
           @toggle="togglePath"
         />
       </div>
       
       <div class="tree-pagination" v-if="hasMore">
         <button @click="loadMore">Load More</button>
       </div>
     </div>
   </template>
   ```
   
   **Features**:
   - Lazy loading (fetch children on expand)
   - Checkbox selection with tri-state (checked, unchecked, partial)
   - Search/filter paths
   - Highlight conflicts (include inside excluded parent)
   - Breadcrumb navigation

2. **Phase 4B: Pattern Generator** (4h)
   
   ```javascript
   function generateResticPatterns(selectedPaths, excludedPaths) {
     // Convert UI selections to Restic include/exclude arrays
     // Handle pattern inheritance
     // Optimize patterns (merge redundant)
     // Validate conflicts
     
     return {
       include: ['/var/www', '/etc/nginx'],
       exclude: ['*.log', '.cache', 'node_modules']
     }
   }
   ```

3. **Phase 4C: Preview Modal** (3h)
   
   ```vue
   <div class="pattern-preview-modal">
     <h3>Pattern Preview</h3>
     
     <div class="patterns">
       <h4>Include Patterns</h4>
       <pre>{{ includePatterns.join('\n') }}</pre>
       
       <h4>Exclude Patterns</h4>
       <pre>{{ excludePatterns.join('\n') }}</pre>
     </div>
     
     <div class="estimated-match">
       <p>Estimated files: ~{{ estimatedFileCount }}</p>
       <p>Estimated size: ~{{ formatBytes(estimatedSize) }}</p>
     </div>
     
     <div class="conflicts" v-if="conflicts.length">
       <h4>⚠️ Conflicts Detected</h4>
       <ul>
         <li v-for="conflict in conflicts">{{ conflict }}</li>
       </ul>
     </div>
   </div>
   ```

4. **Phase 4D: Rule Reordering** (2h)
   - Drag-and-drop to reorder rules
   - Rule priority indicators
   - Rule inheritance visualization

**Tests (TDD-first)**:
- Pattern generation for complex selections
- Conflict detection (include inside exclude)
- Pattern optimization (merge redundant)
- Tri-state checkbox logic
- Lazy loading pagination
- Search filtering

**Estimated Effort**: 17 hours  
**Tests**: +40 new tests (Jest/Vitest for Vue components)

---

### User Story 5 — Agent Sandbox Configuration & Enforcement

**Goal**: Limit agent access to prevent sensitive directory exposure

**Status**: ❌ Not Started

**Priority**: P0 (Security-critical)

**Implementation Plan**:

1. **Phase 5A: Sandbox Engine** (8h)
   
   **New Package**: `agent/pkg/sandbox/`
   
   ```go
   type SandboxEngine struct {
       allowed   []string
       forbidden []string
       maxDepth  int
   }
   
   func NewSandboxEngine(config SandboxConfig) *SandboxEngine
   
   // IsPathAllowed checks if a path is accessible
   func (s *SandboxEngine) IsPathAllowed(path string) (bool, error) {
       // 1. Check if path is in forbidden list
       // 2. Check if path is under allowed roots
       // 3. Check depth limit
       // 4. Handle symlinks (resolve and re-check)
   }
   
   // ValidateCommand checks if a command accesses allowed paths
   func (s *SandboxEngine) ValidateCommand(cmd string) (bool, error) {
       // Parse command for file paths
       // Validate all paths
   }
   ```
   
   **Features**:
   - Path canonicalization (resolve `.`, `..`, symlinks)
   - Recursive depth checking
   - Forbidden path blacklist (e.g., `/etc/shadow`, `/root`)
   - Allowed path whitelist (e.g., `/home`, `/var/www`)
   - Command parsing for hook validation

2. **Phase 5B: Agent Integration** (4h)
   
   ```go
   // In agent/pkg/executor/executor.go
   
   type Executor struct {
       sandbox *sandbox.SandboxEngine
   }
   
   func (e *Executor) ExecuteBackup(task Task) error {
       // Validate all include paths
       for _, path := range task.IncludePaths {
           if allowed, _ := e.sandbox.IsPathAllowed(path); !allowed {
               return fmt.Errorf("path %s not allowed by sandbox", path)
           }
       }
       
       // Execute backup
   }
   ```

3. **Phase 5C: Orchestrator Validation** (3h)
   
   ```go
   // In internal/api/policy.go
   
   func (h *PolicyHandler) CreatePolicy(c *gin.Context) {
       var policy Policy
       c.BindJSON(&policy)
       
       // Validate sandbox config
       if policy.SandboxConfig != nil {
           validator := NewPolicyValidator(h.db)
           if err := validator.ValidateSandbox(policy); err != nil {
               c.JSON(400, gin.H{"error": err.Error()})
               return
           }
       }
   }
   ```

4. **Phase 5D: Preflight Integration** (Epic 17 dependency) (2h)
   - Hook into preflight validation
   - Run sandbox checks before task dispatch
   - Alert on policy violations

**Tests (TDD-first)**:
- Path allowed vs forbidden
- Symlink resolution
- Depth limit enforcement
- Command parsing (hook validation)
- Canonical path handling
- Edge cases (root paths, relative paths)

**Estimated Effort**: 17 hours  
**Tests**: +50 new tests

---

### User Story 6 — Repository Credentials & Certificate Management

**Goal**: Securely store and deliver repo passwords and certificates

**Status**: ❌ Not Started

**Priority**: P0 (Security-critical)

**Implementation Plan**:

1. **Phase 6A: Credential Model** (4h)
   
   **New Model**: `internal/store/credential.go`
   
   ```go
   type Credential struct {
       ID               uuid.UUID `gorm:"primaryKey" json:"id"`
       TenantID         uuid.UUID `gorm:"not null;index" json:"tenant_id"`
       Name             string    `gorm:"type:varchar(255);not null" json:"name"`
       Type             string    `gorm:"type:varchar(50);not null" json:"type"` // password, cert, aws, gcs
       
       // Encrypted at rest
       PasswordHash     *string   `gorm:"type:text" json:"-"` // bcrypt hash
       CertificatePEM   *string   `gorm:"type:text" json:"-"`
       CertificateKeyPEM *string   `gorm:"type:text" json:"-"`
       CAChainPEM       *string   `gorm:"type:text" json:"-"`
       
       // AWS/GCS credentials
       AccessKeyID      *string   `gorm:"type:text" json:"-"`
       SecretAccessKey  *string   `gorm:"type:text" json:"-"` // Encrypted
       
       // Metadata
       ExpiresAt        *time.Time `json:"expires_at,omitempty"`
       CreatedAt        time.Time  `json:"created_at"`
       UpdatedAt        time.Time  `json:"updated_at"`
   }
   ```
   
   **Migration 008**: Create credentials table

2. **Phase 6B: Encryption Service** (6h)
   
   **New Package**: `internal/crypto/`
   
   ```go
   type EncryptionService struct {
       masterKey []byte
   }
   
   // EncryptSecret encrypts data with AES-256-GCM
   func (s *EncryptionService) EncryptSecret(plaintext string) (string, error)
   
   // DecryptSecret decrypts data
   func (s *EncryptionService) DecryptSecret(ciphertext string) (string, error)
   
   // GenerateToken creates time-limited token for credential access
   func (s *EncryptionService) GenerateToken(credentialID uuid.UUID, ttl time.Duration) (string, error)
   
   // ValidateToken checks token validity and extracts credential ID
   func (s *EncryptionService) ValidateToken(token string) (uuid.UUID, error)
   ```
   
   **Features**:
   - AES-256-GCM encryption
   - Master key from environment variable
   - Key rotation support
   - Time-limited access tokens (JWT)

3. **Phase 6C: Credential API** (4h)
   
   ```go
   // POST /api/v1/credentials
   func (h *CredentialHandler) CreateCredential(c *gin.Context)
   
   // GET /api/v1/credentials
   func (h *CredentialHandler) ListCredentials(c *gin.Context)
   
   // DELETE /api/v1/credentials/{id}
   func (h *CredentialHandler) DeleteCredential(c *gin.Context)
   
   // POST /api/v1/credentials/{id}/token
   func (h *CredentialHandler) GenerateAccessToken(c *gin.Context)
   ```

4. **Phase 6D: Agent Token Validation** (3h)
   
   ```go
   // In agent/pkg/client/orchestrator.go
   
   func (c *OrchestratorClient) RetrieveCredentials(token string) (*Credentials, error) {
       // POST /api/v1/credentials/validate with token
       // Receive decrypted credentials (one-time use)
   }
   ```

5. **Phase 6E: Certificate Validation** (2h)
   - Parse PEM certificates
   - Check expiry dates
   - Validate CA chain
   - Alert on near-expiry (30 days)

**Tests (TDD-first)**:
- Encryption/decryption roundtrip
- Token generation and validation
- Token expiry
- Certificate parsing
- Certificate expiry detection
- Credential CRUD operations
- Access token lifecycle

**Estimated Effort**: 19 hours  
**Tests**: +60 new tests

---

### User Story 7 — Pre/Post Hook Integration

**Goal**: Assign hooks to policies for execution before/after backups

**Status**: ⚠️ 30% Complete (Epic 14 plugin system exists)

**Dependencies**:
- Epic 14 plugin/hook template library
- Sandbox Engine (US #5) - hooks must respect sandbox

**Implementation Plan**:

1. **Phase 7A: Hook Model** (Epic 14 scope) (2h)
   
   **Assumption**: Epic 14 provides:
   ```go
   type HookTemplate struct {
       ID          uuid.UUID
       Name        string
       Description string
       Command     string
       Parameters  JSONB
       Timeout     time.Duration
   }
   ```

2. **Phase 7B: Policy-Hook Integration** (4h)
   
   ```go
   // In Policy model (already planned in Phase 1A)
   PreHooks  JSONB `json:"pre_hooks,omitempty"`  // Array of hook IDs
   PostHooks JSONB `json:"post_hooks,omitempty"` // Array of hook IDs
   ```
   
   **UI Changes**: `frontend/src/App.vue`
   - Add hook selection to policy create/edit modal
   - Fetch available hooks from Epic 14 API
   - Display hook parameters
   - Validate hook selection

3. **Phase 7C: Hook Execution in Task** (3h)
   
   ```go
   // In agent/pkg/executor/executor.go
   
   func (e *Executor) ExecuteBackup(task Task) error {
       // Execute pre-hooks
       for _, hookID := range task.PreHooks {
           if err := e.executeHook(hookID); err != nil {
               return fmt.Errorf("pre-hook failed: %w", err)
           }
       }
       
       // Execute backup
       if err := e.runResticBackup(task); err != nil {
           return err
       }
       
       // Execute post-hooks
       for _, hookID := range task.PostHooks {
           if err := e.executeHook(hookID); err != nil {
               // Log but don't fail backup
               e.logger.Warn("post-hook failed", "error", err)
           }
       }
   }
   ```

4. **Phase 7D: Hook Validation (Preflight)** (Epic 17 scope) (2h)
   - Validate hook exists
   - Validate hook command respects sandbox
   - Validate hook parameters

**Tests (TDD-first)**:
- Hook selection in policy
- Pre-hook execution
- Post-hook execution
- Hook failure handling
- Hook sandbox validation
- Hook timeout enforcement

**Estimated Effort**: 11 hours (4h in Epic 16, 7h deferred to Epic 14/17)  
**Tests**: +25 new tests

---

### User Story 8 — Policy–Agent Link Management UI

**Goal**: Easily assign/reassign policies to agents

**Status**: ✅ 90% Complete

**What's Done**:
- ✅ Assign modal with agent list
- ✅ Detach modal with assigned agents
- ✅ Assignment API (POST/DELETE)
- ✅ Policy health status (enabled/disabled)

**What's Missing**:
- ❌ Policy validation status indicator (valid/invalid)
- ❌ Preflight check on assignment
- ❌ Bulk assignment UI

**Implementation Plan**:

1. **Phase 8A: Policy Validation Status** (2h)
   
   Add validation status to policy:
   ```go
   type Policy struct {
       // ...
       ValidationStatus string `gorm:"type:varchar(50)" json:"validation_status"` // valid, invalid, warning
       ValidationErrors JSONB  `gorm:"serializer:json" json:"validation_errors,omitempty"`
   }
   ```
   
   Background job validates all policies every hour:
   - Check sandbox conflicts
   - Check credential existence
   - Check hook validity
   - Update validation_status

2. **Phase 8B: UI Indicators** (2h)
   
   ```vue
   <div class="policy-card">
     <div class="policy-status">
       <span v-if="policy.validationStatus === 'valid'" class="badge badge-success">✓ Valid</span>
       <span v-if="policy.validationStatus === 'invalid'" class="badge badge-error">✗ Invalid</span>
       <span v-if="policy.validationStatus === 'warning'" class="badge badge-warning">⚠ Warning</span>
     </div>
     
     <div v-if="policy.validationErrors" class="validation-errors">
       <ul>
         <li v-for="error in policy.validationErrors">{{ error }}</li>
       </ul>
     </div>
   </div>
   ```

3. **Phase 8C: Bulk Assignment** (3h)
   
   ```vue
   <div class="bulk-assign-modal">
     <h3>Assign Policy to Multiple Agents</h3>
     
     <div class="agent-selection">
       <label v-for="agent in agents">
         <input type="checkbox" v-model="selectedAgents" :value="agent.id" />
         {{ agent.hostname }}
       </label>
     </div>
     
     <button @click="bulkAssign">Assign to {{ selectedAgents.length }} agents</button>
   </div>
   ```

**Estimated Effort**: 7 hours  
**Tests**: +15 new tests

---

## Database Schema Extensions

### New Tables

#### 1. Credentials Table (US #6)

```sql
CREATE TABLE credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('password', 'cert', 'aws', 'gcs')),
    
    -- Encrypted fields
    password_hash TEXT,
    certificate_pem TEXT,
    certificate_key_pem TEXT,
    ca_chain_pem TEXT,
    access_key_id TEXT,
    secret_access_key TEXT,  -- Encrypted with AES-256-GCM
    
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uk_credentials_name UNIQUE (tenant_id, name)
);

CREATE INDEX idx_credentials_tenant ON credentials(tenant_id);
CREATE INDEX idx_credentials_type ON credentials(type);
CREATE INDEX idx_credentials_expires ON credentials(expires_at);
```

#### 2. Agent Sandbox Configs (US #5)

**Option A**: Store in Agent model as JSONB (simpler, already implemented)

**Option B**: Separate table (more flexible):

```sql
CREATE TABLE agent_sandboxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    allowed_paths JSONB NOT NULL,      -- Array of strings
    forbidden_paths JSONB NOT NULL,    -- Array of strings
    max_depth INTEGER NOT NULL DEFAULT 20,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uk_agent_sandbox UNIQUE (agent_id)
);
```

**Decision**: Use JSONB in Agent model for simplicity (Option A)

### Column Additions to Existing Tables

#### Policies Table Extensions

```sql
ALTER TABLE policies
    ADD COLUMN sandbox_config JSONB,
    ADD COLUMN credentials_id UUID REFERENCES credentials(id) ON DELETE RESTRICT,
    ADD COLUMN pre_hooks JSONB,         -- Array of hook IDs
    ADD COLUMN post_hooks JSONB,        -- Array of hook IDs
    ADD COLUMN validation_status VARCHAR(50) DEFAULT 'valid',
    ADD COLUMN validation_errors JSONB,
    ADD COLUMN policy_version INTEGER DEFAULT 1;

CREATE INDEX idx_policies_credentials ON policies(credentials_id);
CREATE INDEX idx_policies_validation_status ON policies(validation_status);
```

#### Agents Table Extensions

```sql
ALTER TABLE agents
    ADD COLUMN sandbox_config JSONB;  -- SandboxConfig: {allowed, forbidden, maxDepth}
```

#### Tasks Table Extensions

```sql
ALTER TABLE tasks
    ADD COLUMN credentials_token TEXT,  -- Time-limited JWT
    ADD COLUMN pre_hooks JSONB,
    ADD COLUMN post_hooks JSONB,
    ADD COLUMN sandbox_config JSONB,
    ADD COLUMN policy_version INTEGER;

CREATE INDEX idx_tasks_policy_version ON tasks(policy_version);
```

### Migrations

**Migration 007: EPIC 16 Phase 1 - Policy Extensions**
- Add sandbox_config, credentials_id, pre_hooks, post_hooks to policies
- Add validation_status, validation_errors, policy_version to policies
- Add sandbox_config to agents

**Migration 008: EPIC 16 Phase 2 - Credentials**
- Create credentials table
- Add foreign key from policies to credentials

**Migration 009: EPIC 16 Phase 3 - Task Extensions**
- Add credentials_token, hooks, sandbox_config to tasks

---

## Architecture Changes

### New Components

#### 1. Filesystem Service (Agent)

**Package**: `agent/pkg/filesystem/`

**Responsibilities**:
- Serve filesystem tree API
- Enforce sandbox restrictions
- Handle large directory pagination
- Detect file types and permissions

**API Endpoints** (agent-only):
- `GET /agent/fs/root` - Get root nodes
- `GET /agent/fs/tree?path=/var&depth=3&hidden=false` - Get directory tree

#### 2. Sandbox Engine (Agent)

**Package**: `agent/pkg/sandbox/`

**Responsibilities**:
- Validate paths against allowed/forbidden lists
- Enforce max depth traversal
- Resolve symlinks and canonical paths
- Parse commands for path references

**Used By**:
- FilesystemService (browsing)
- Executor (backup/check/prune)
- Hook executor

#### 3. Credential Store (Orchestrator)

**Package**: `internal/credential/`

**Responsibilities**:
- CRUD for credentials
- Encrypt/decrypt secrets
- Generate time-limited access tokens
- Validate certificates

**API Endpoints**:
- `POST /api/v1/credentials` - Create credential
- `GET /api/v1/credentials` - List credentials
- `DELETE /api/v1/credentials/{id}` - Delete credential
- `POST /api/v1/credentials/{id}/token` - Generate access token
- `POST /api/v1/credentials/validate` - Validate token and retrieve secrets

#### 4. Policy Validator (Orchestrator)

**Package**: `internal/policy/`

**Responsibilities**:
- Validate policy schema
- Detect conflicts (sandbox, hooks, retention)
- Check credential references
- Validate cron expressions
- Background validation job

**Used By**:
- PolicyHandler (on create/update)
- Scheduler (before task creation)
- Background job (periodic validation)

#### 5. Hook Integration Layer (Orchestrator)

**Package**: `internal/hooks/` (Epic 14 may define this)

**Responsibilities**:
- Fetch hook templates from Epic 14 system
- Validate hook parameters
- Attach hooks to tasks

**API Endpoints**:
- `GET /api/v1/hooks` - List available hooks (Epic 14)
- `GET /api/v1/hooks/{id}` - Get hook details (Epic 14)

---

## Testing Strategy (TDD-First)

### Test Infrastructure

#### 1. Virtual Filesystem Layer

**Purpose**: Simulate agent filesystems for testing without real files

**Implementation**: `agent/pkg/filesystem/testfs/`

```go
type VirtualFS struct {
    nodes map[string]*FSNode
}

func NewVirtualFS() *VirtualFS
func (vfs *VirtualFS) AddDir(path string, mode os.FileMode)
func (vfs *VirtualFS) AddFile(path string, size int64, mode os.FileMode)
func (vfs *VirtualFS) AddSymlink(path, target string)
```

**Used By**:
- Filesystem service tests
- Sandbox engine tests
- Large directory pagination tests

#### 2. Synthetic Directory Structures

**Purpose**: Pre-built test fixtures for common scenarios

**Examples**:
- `fixtures/forbidden_access/` - Directories with /etc/shadow, /root
- `fixtures/large_dir/` - 100,000 file directory for pagination
- `fixtures/deep_nesting/` - 50-level deep directory for depth limits
- `fixtures/symlinks/` - Complex symlink chains

#### 3. Hook Command Parser Tests

**Purpose**: Validate that sandbox correctly parses commands for path references

**Test Cases**:
```bash
# Simple path reference
mysqldump --result-file=/tmp/dump.sql mydb
# → Extract: /tmp/dump.sql → validate against sandbox

# Multiple paths
tar -czf /backup/archive.tar.gz /var/www /etc/nginx
# → Extract: /backup/archive.tar.gz, /var/www, /etc/nginx → validate all

# Piped commands
pg_dump mydb | gzip > /backup/dump.sql.gz
# → Extract: /backup/dump.sql.gz → validate
```

#### 4. Credential Encryption Tests

**Purpose**: Ensure encryption/decryption works correctly

**Test Cases**:
- Roundtrip encryption (encrypt → decrypt → match original)
- Key rotation (re-encrypt with new key)
- Token expiry (generate → wait → validate = expired)
- Invalid token (tampered data → error)

#### 5. Conflict Detection Tests

**Purpose**: Validate PolicyValidator catches all conflicts

**Test Scenarios**:
```go
// Conflict: Include path in forbidden directory
policy := Policy{
    IncludePaths: []string{"/etc/shadow"},
    SandboxConfig: SandboxConfig{
        Forbidden: []string{"/etc/shadow"},
    },
}
// → Expected: ValidationError("include path /etc/shadow is forbidden by sandbox")

// Conflict: Hook accessing forbidden path
policy := Policy{
    PreHooks: []string{"hook-mysql-dump"},  // Command: mysqldump > /root/dump.sql
    SandboxConfig: SandboxConfig{
        Forbidden: []string{"/root"},
    },
}
// → Expected: ValidationError("pre-hook mysql-dump accesses forbidden path /root")
```

### Test Counts by User Story

| User Story | Backend Tests | Frontend Tests | Total |
|------------|--------------|----------------|-------|
| US #1 (Policy CRUD) | 45 | 0 | 45 |
| US #2 (Agent Policy Application) | 30 | 0 | 30 |
| US #3 (Filesystem Browser) | 35 | 0 | 35 |
| US #4 (Visual Rule Builder) | 10 | 30 | 40 |
| US #5 (Sandbox Engine) | 50 | 0 | 50 |
| US #6 (Credentials) | 60 | 10 | 70 |
| US #7 (Hook Integration) | 20 | 5 | 25 |
| US #8 (Link Management UI) | 5 | 10 | 15 |
| **Total** | **255** | **55** | **310** |

**Grand Total**: **310 new tests** (679 existing + 310 = **989 tests**)

---

## Implementation Phases

### Phase 1: Foundation (Week 1-2) — 40 hours

**Goal**: Extend data models and add validation

**Deliverables**:
- ✅ Policy model extensions (sandbox, credentials, hooks)
- ✅ Credential model + table
- ✅ PolicyValidator service
- ✅ Conflict detection
- ✅ Migration 007 + 008
- ✅ 105 tests passing

**User Stories**: US #1 (complete)

---

### Phase 2: Security Infrastructure (Week 3-4) — 45 hours

**Goal**: Implement sandbox engine and credential encryption

**Deliverables**:
- ✅ SandboxEngine (agent-side)
- ✅ EncryptionService (orchestrator-side)
- ✅ Credential API (CRUD + token generation)
- ✅ Agent sandbox validation
- ✅ Certificate validation
- ✅ Migration 009
- ✅ 110 tests passing

**User Stories**: US #5 (complete), US #6 (complete), US #2 (partial)

---

### Phase 3: Filesystem Browser (Week 5) — 25 hours

**Goal**: Enable agent filesystem browsing via API

**Deliverables**:
- ✅ FilesystemService (agent)
- ✅ Orchestrator proxy endpoint
- ✅ Pagination for large directories
- ✅ Caching (24h TTL)
- ✅ 35 tests passing

**User Stories**: US #3 (complete)

---

### Phase 4: Visual Rule Builder UI (Week 6-7) — 40 hours

**Goal**: Build interactive filesystem tree and pattern generator

**Deliverables**:
- ✅ FilesystemTree.vue component
- ✅ Lazy loading + search
- ✅ Pattern generator
- ✅ Preview modal
- ✅ Conflict highlighting
- ✅ 40 tests passing (Vitest)

**User Stories**: US #4 (complete)

---

### Phase 5: Hook Integration (Week 8) — 15 hours

**Goal**: Connect policies to Epic 14 hook system

**Deliverables**:
- ✅ Hook selection in policy UI
- ✅ Hook execution in agent
- ✅ Hook validation in PolicyValidator
- ✅ 25 tests passing

**User Stories**: US #7 (complete), US #2 (complete)

**Dependencies**: Epic 14 hook template API must exist

---

### Phase 6: Polish & Completion (Week 9) — 15 hours

**Goal**: Finish remaining UI features and documentation

**Deliverables**:
- ✅ Policy validation status UI
- ✅ Bulk assignment UI
- ✅ Background validation job
- ✅ Documentation updates
- ✅ 15 tests passing

**User Stories**: US #8 (complete)

---

## Total Effort Estimation

| Phase | Duration | Tests | Key Deliverables |
|-------|----------|-------|------------------|
| Phase 1 | 40h | +105 | Policy extensions, validation, credentials model |
| Phase 2 | 45h | +110 | Sandbox engine, encryption, credential API |
| Phase 3 | 25h | +35 | Filesystem browser API |
| Phase 4 | 40h | +40 | Visual rule builder UI |
| Phase 5 | 15h | +25 | Hook integration |
| Phase 6 | 15h | +15 | Polish & completion |
| **Total** | **180h** | **+310** | **All 8 user stories complete** |

**Timeline**: 9 weeks (full-time) or 18 weeks (part-time)

**Post-Epic Test Count**: 679 + 310 = **989 tests**

---

## Dependencies

### Inbound Dependencies (EPIC 16 depends on)

1. **Epic 14 (Plugins/Hooks)** - Partial dependency
   - Hook template model
   - Hook API endpoints
   - Hook execution framework
   - **Workaround**: Can implement Epic 16 without hooks, add later

2. **Epic 15 (Concurrency Control)** - ✅ Complete
   - Agent concurrency limits
   - Task queue management

### Outbound Dependencies (Other epics depend on EPIC 16)

1. **Epic 17 (Preflight Validation)** - Strong dependency
   - Requires PolicyValidator
   - Requires SandboxEngine
   - Requires policy versioning

2. **Epic 18 (Smart Triggering)** - Strong dependency
   - Requires policy engine
   - Requires policy validation

3. **Epic 19 (Forecasting)** - Moderate dependency
   - Requires policy analytics
   - Requires retention rules

4. **Epic 20 (DR + Restore)** - Strong dependency
   - Requires credential management
   - Requires repository configuration

5. **Epic 20B (Policy Wizard)** - Strong dependency
   - Requires visual rule builder
   - Requires filesystem browser

---

## Risks & Mitigations

### Risk 1: Epic 14 Hook System Not Ready

**Impact**: Cannot implement US #7 (Hook Integration)

**Probability**: Medium

**Mitigation**:
- Defer US #7 to later phase
- Implement placeholder hook model
- Complete other 7 user stories first

### Risk 2: Encryption Key Management Complexity

**Impact**: Credential storage security concerns

**Probability**: Low

**Mitigation**:
- Use battle-tested crypto libraries (Go's crypto/aes)
- Document key rotation procedure
- Support external key management (HashiCorp Vault) in future

### Risk 3: Filesystem Browser Performance

**Impact**: Slow UI on large directories

**Probability**: Medium

**Mitigation**:
- Implement pagination (limit 1000 per page)
- Add caching with 24h TTL
- Add search/filter to reduce results
- Lazy loading in UI

### Risk 4: Sandbox Bypass Vulnerabilities

**Impact**: Security breach, sensitive data exposed

**Probability**: Low

**Mitigation**:
- Comprehensive security testing
- Symlink resolution
- Path canonicalization
- Security audit before production release

---

## Success Criteria

### Functional Requirements

- [ ] All 8 user stories implemented and tested
- [ ] 310 new tests passing (989 total)
- [ ] All database migrations applied successfully
- [ ] UI components functional and accessible
- [ ] API documentation updated (Swagger)

### Performance Requirements

- [ ] Filesystem tree loads < 2s for 10k files
- [ ] Policy validation completes < 500ms
- [ ] Credential encryption/decryption < 100ms
- [ ] Pattern preview generates < 1s

### Security Requirements

- [ ] Credentials encrypted at rest (AES-256-GCM)
- [ ] Access tokens expire after 1 hour
- [ ] Sandbox prevents access to forbidden paths
- [ ] Certificate validation catches expired certs

### Usability Requirements

- [ ] Visual rule builder intuitive (user testing)
- [ ] Conflict detection provides clear error messages
- [ ] Policy validation shows actionable warnings
- [ ] Bulk assignment saves time vs. individual assignment

---

## Post-Epic 16 Capabilities

After completion, the system will support:

✅ **Centralized Policy Management**
- Create policies with visual rule builder
- Assign policies to multiple agents
- Validate policies before deployment

✅ **Secure Credential Handling**
- Store repository passwords encrypted
- Manage TLS certificates with expiry alerts
- Time-limited credential delivery to agents

✅ **Agent Sandbox Security**
- Prevent backup of sensitive directories
- Enforce path access restrictions
- Validate hooks against sandbox

✅ **Visual Filesystem Browser**
- Browse agent filesystems remotely
- Select include/exclude paths visually
- Preview Restic patterns before saving

✅ **Pre/Post Hook Integration**
- Database dumps before backup
- Service pause/resume around backup
- Custom scripts with validation

✅ **Policy Conflict Detection**
- Catch sandbox violations
- Validate retention rules
- Alert on invalid schedules

✅ **Foundation for Advanced Epics**
- Epic 17: Preflight validation framework ready
- Epic 18: Policy engine ready for smart triggering
- Epic 19: Policy analytics data available
- Epic 20: Credential management for restore operations

---

## Next Steps

1. **Review & Approval** (This document)
   - Stakeholder review of scope
   - Approve 180-hour effort estimate
   - Confirm 9-week timeline

2. **Environment Setup**
   - Set up test infrastructure (virtual FS, fixtures)
   - Configure encryption keys
   - Prepare staging environment

3. **Phase 1 Kickoff**
   - Create feature branch: `epic16-phase1-policy-extensions`
   - Start with TDD: Write first test for policy sandbox config
   - Implement PolicyValidator service

4. **Documentation**
   - Update architecture.md with new components
   - Create API documentation for new endpoints
   - Write user guide for visual rule builder

---

**Status**: 📋 Ready for implementation  
**Next Review**: After Phase 1 completion (Week 2)  
**Last Updated**: 2025-11-26
