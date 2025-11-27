# EPIC 16 — Implementation Roadmap

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           EPIC 16 SUPER-EPIC                                │
│         Unified Multi-Agent Backup Policy Orchestration Framework           │
│                                                                             │
│  Timeline: 9 Weeks (180h) | Tests: +310 (989 total) | Phases: 6           │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ WEEK 1-2: PHASE 1 — FOUNDATION (40h)                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│ GOAL: Extend data models and add policy validation                         │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 1A: Extend Policy Model (4h)                                  │   │
│ │ • Add sandbox_config, credentials_id, pre_hooks, post_hooks         │   │
│ │ • SandboxConfig struct definition                                   │   │
│ │ • Schema validation                                                 │   │
│ │ Tests: +15                                                          │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 1B: Conflict Detection Service (6h)                           │   │
│ │ • PolicyValidator.ValidatePolicy()                                  │   │
│ │ • Check includes vs sandbox                                         │   │
│ │ • Check hooks vs sandbox                                            │   │
│ │ • Validate schedule, retention, credentials                         │   │
│ │ Tests: +30                                                          │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 1C: Database Migration 007 (2h)                               │   │
│ │ • ALTER policies ADD COLUMN sandbox_config, credentials_id, etc.    │   │
│ │ • ALTER agents ADD COLUMN sandbox_config                            │   │
│ │ • Backward compatibility (NULL = unrestricted)                      │   │
│ │ Tests: +5 (migration tests)                                         │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ DELIVERABLES: US #1 Complete | Tests: +50 (729 total)                     │
│ BRANCH: epic16-phase1-policy-extensions                                    │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ WEEK 3-4: PHASE 2 — SECURITY INFRASTRUCTURE (45h)                          │
├─────────────────────────────────────────────────────────────────────────────┤
│ GOAL: Implement sandbox engine and credential encryption                   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 2A: Credential Model (4h)                                     │   │
│ │ • internal/store/credential.go                                      │   │
│ │ • Encrypted fields (password, cert, AWS keys)                       │   │
│ │ • Migration 008: CREATE TABLE credentials                           │   │
│ │ Tests: +15                                                          │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 2B: Encryption Service (6h)                                   │   │
│ │ • internal/crypto/encryption.go                                     │   │
│ │ • AES-256-GCM encryption/decryption                                 │   │
│ │ • JWT token generation (1h TTL)                                     │   │
│ │ • Token validation                                                  │   │
│ │ Tests: +20 (roundtrip, expiry, rotation)                            │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 2C: Credential API (4h)                                       │   │
│ │ • POST /api/v1/credentials                                          │   │
│ │ • GET /api/v1/credentials                                           │   │
│ │ • DELETE /api/v1/credentials/{id}                                   │   │
│ │ • POST /api/v1/credentials/{id}/token                               │   │
│ │ Tests: +15                                                          │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 2D: Sandbox Engine (8h)                                       │   │
│ │ • agent/pkg/sandbox/engine.go                                       │   │
│ │ • IsPathAllowed() with whitelist/blacklist                          │   │
│ │ • ValidateCommand() for hooks                                       │   │
│ │ • Symlink resolution, depth checks                                  │   │
│ │ Tests: +35 (paths, symlinks, depth, commands)                       │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 2E: Agent Integration (4h)                                    │   │
│ │ • agent/pkg/executor/executor.go                                    │   │
│ │ • Validate task paths against sandbox                               │   │
│ │ • Validate hook commands                                            │   │
│ │ Tests: +10                                                          │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 2F: Certificate Validation (2h)                               │   │
│ │ • Parse PEM certificates                                            │   │
│ │ • Check expiry dates (alert at 30 days)                             │   │
│ │ • Validate CA chain                                                 │   │
│ │ Tests: +10                                                          │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ DELIVERABLES: US #5, US #6, US #2 (partial) | Tests: +105 (834 total)     │
│ BRANCH: epic16-phase2-security                                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ WEEK 5: PHASE 3 — FILESYSTEM BROWSER (25h)                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│ GOAL: Enable agent filesystem browsing via API                             │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 3A: Filesystem Service (6h)                                   │   │
│ │ • agent/pkg/filesystem/service.go                                   │   │
│ │ • GET /agent/fs/root                                                │   │
│ │ • GET /agent/fs/tree?path=/var&depth=3&hidden=false                 │   │
│ │ • FSNode struct (path, name, type, size, children)                  │   │
│ │ Tests: +20 (sandbox, pagination, symlinks)                          │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 3B: Orchestrator Proxy (3h)                                   │   │
│ │ • GET /api/v1/agents/{id}/filesystem/tree                           │   │
│ │ • Forward request to agent                                          │   │
│ │ • Cache response (24h TTL)                                          │   │
│ │ Tests: +10                                                          │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 3C: Large Directory Handling (2h)                             │   │
│ │ • Cursor-based pagination                                           │   │
│ │ • Timeout protection (30s max)                                      │   │
│ │ • Streaming for very large dirs                                     │   │
│ │ Tests: +5 (>10k files)                                              │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ DELIVERABLES: US #3 Complete | Tests: +35 (869 total)                     │
│ BRANCH: epic16-phase3-filesystem                                           │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ WEEK 6-7: PHASE 4 — VISUAL RULE BUILDER UI (40h)                           │
├─────────────────────────────────────────────────────────────────────────────┤
│ GOAL: Build interactive filesystem tree and pattern generator              │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 4A: Tree Component (8h)                                       │   │
│ │ • frontend/src/components/FilesystemTree.vue                        │   │
│ │ • Lazy loading (fetch children on expand)                           │   │
│ │ • Tri-state checkboxes (checked/unchecked/partial)                  │   │
│ │ • Search/filter paths                                               │   │
│ │ • Conflict highlighting                                             │   │
│ │ Tests: +15 (Vitest)                                                 │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 4B: Pattern Generator (4h)                                    │   │
│ │ • generateResticPatterns(selections)                                │   │
│ │ • Convert UI selections to include/exclude arrays                   │   │
│ │ • Pattern optimization (merge redundant)                            │   │
│ │ • Validate conflicts                                                │   │
│ │ Tests: +10 (pattern generation, optimization)                       │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 4C: Preview Modal (3h)                                        │   │
│ │ • Pattern preview with syntax highlighting                          │   │
│ │ • Estimated file count                                              │   │
│ │ • Estimated size                                                    │   │
│ │ • Conflict warnings                                                 │   │
│ │ Tests: +10 (preview calculations)                                   │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 4D: Rule Reordering (2h)                                      │   │
│ │ • Drag-and-drop rules                                               │   │
│ │ • Priority indicators                                               │   │
│ │ • Inheritance visualization                                         │   │
│ │ Tests: +5                                                           │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ DELIVERABLES: US #4 Complete | Tests: +40 (909 total)                     │
│ BRANCH: epic16-phase4-rule-builder                                         │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ WEEK 8: PHASE 5 — HOOK INTEGRATION (15h)                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ GOAL: Connect policies to Epic 14 hook system                              │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 5A: Hook Selection UI (3h)                                    │   │
│ │ • Add hook selection to policy modal                                │   │
│ │ • Fetch hooks from Epic 14 API                                      │   │
│ │ • Display hook parameters                                           │   │
│ │ Tests: +5 (Vue component)                                           │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 5B: Hook Execution in Task (3h)                               │   │
│ │ • agent/pkg/executor/executor.go                                    │   │
│ │ • Execute pre-hooks before backup                                   │   │
│ │ • Execute post-hooks after backup                                   │   │
│ │ • Handle hook failures                                              │   │
│ │ Tests: +15 (pre/post execution, failures)                           │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 5C: Hook Validation (2h)                                      │   │
│ │ • PolicyValidator.ValidateHooks()                                   │   │
│ │ • Check hook exists                                                 │   │
│ │ • Check hook command vs sandbox                                     │   │
│ │ Tests: +5 (validation tests)                                        │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ DELIVERABLES: US #7, US #2 Complete | Tests: +25 (934 total)              │
│ BRANCH: epic16-phase5-hooks                                                │
│                                                                             │
│ NOTE: Requires Epic 14 hook template API to exist                          │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ WEEK 9: PHASE 6 — POLISH & COMPLETION (15h)                                │
├─────────────────────────────────────────────────────────────────────────────┤
│ GOAL: Finish remaining UI features and documentation                       │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 6A: Policy Validation Status (2h)                             │   │
│ │ • Add validation_status column to policies                          │   │
│ │ • Background job to validate all policies (hourly)                  │   │
│ │ • Update validation_status based on checks                          │   │
│ │ Tests: +5                                                           │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 6B: UI Indicators (2h)                                        │   │
│ │ • Show validation status badges (valid/invalid/warning)             │   │
│ │ • Display validation errors in tooltip                              │   │
│ │ Tests: +5 (Vue component)                                           │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Phase 6C: Bulk Assignment (3h)                                      │   │
│ │ • Bulk assignment modal                                             │   │
│ │ • Multi-select agents                                               │   │
│ │ • Batch API call                                                    │   │
│ │ Tests: +5                                                           │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ DELIVERABLES: US #8 Complete | Tests: +15 (949 total)                     │
│ BRANCH: epic16-phase6-polish                                               │
│                                                                             │
│ FINAL: Merge all phases to main | Update docs | 989 TESTS TOTAL           │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                           COMPLETION CRITERIA                               │
├─────────────────────────────────────────────────────────────────────────────┤
│ ✅ All 8 user stories implemented and tested                               │
│ ✅ 310 new tests passing (989 total)                                       │
│ ✅ 3 migrations applied (007-009)                                          │
│ ✅ 4 new services operational (Validator, Sandbox, Encryption, Filesystem) │
│ ✅ UI components functional and accessible                                 │
│ ✅ API documentation updated (Swagger)                                     │
│ ✅ Performance benchmarks met (FS <2s, validation <500ms, encryption <100ms)│
│ ✅ Security audit passed (encrypted credentials, sandbox enforcement)      │
│ ✅ User acceptance testing completed (visual rule builder, bulk assign)    │
│ ✅ Documentation complete (epic16-status.md, architecture.md updates)      │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                           DEPENDENCY TREE                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  EPIC 15 (Complete) ──┐                                                    │
│                       ↓                                                     │
│  EPIC 14 (Partial) ───→  EPIC 16 (This Epic)  ───→  EPIC 17 (Preflight)   │
│                                   ↓                                         │
│                                   ├────────────────→  EPIC 18 (Smart)      │
│                                   ├────────────────→  EPIC 19 (Forecast)   │
│                                   ├────────────────→  EPIC 20 (DR/Restore) │
│                                   └────────────────→  EPIC 20B (Wizard)    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                              RISK MATRIX                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  HIGH IMPACT   │                          │ Sandbox Bypass (Security)     │
│                │                          │ Mitigation: Audit, tests      │
│                ├──────────────────────────┼───────────────────────────────┤
│                │ Epic 14 Not Ready        │                               │
│  MED IMPACT    │ Mitigation: Defer US #7  │ FS Browser Slow               │
│                │                          │ Mitigation: Cache, paginate   │
│                ├──────────────────────────┼───────────────────────────────┤
│  LOW IMPACT    │                          │ Encryption Key Complexity     │
│                │                          │ Mitigation: Use crypto/aes    │
│                │                          │                               │
│                └──────────────────────────┴───────────────────────────────┘
│                    LOW              MEDIUM              HIGH
│                         PROBABILITY
└─────────────────────────────────────────────────────────────────────────────┘

Last Updated: 2025-11-26
Full Documentation: /docs/epic16-status.md
Quick Reference: /docs/epic16-summary.md
