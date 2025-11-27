# EPIC 16 — Quick Reference Summary

## 📊 At a Glance

| Metric | Value |
|--------|-------|
| **Total Effort** | 180 hours |
| **Timeline** | 9 weeks (full-time) |
| **New Tests** | +310 tests |
| **Post-Epic Tests** | 989 total |
| **User Stories** | 8 |
| **Phases** | 6 |
| **New Models** | 1 (Credential) |
| **Migrations** | 3 (007-009) |
| **New Services** | 4 |
| **Status** | 📋 Planning Complete |

---

## 🎯 What is EPIC 16?

A **super-epic** that consolidates:

1. **Multi-agent orchestration** (original scope)
2. **Filesystem browser** (visual path selection)
3. **Visual rule builder** (include/exclude patterns)
4. **Agent sandbox** (security enforcement)
5. **Credential management** (passwords, certs - from Epic 10)
6. **Hook integration** (pre/post backup - with Epic 14)

**Strategic Goal**: Build the foundation for all advanced backup features (Epics 17-21)

---

## 📋 8 User Stories

| # | User Story | Status | Effort | Tests |
|---|------------|--------|--------|-------|
| 1 | Backup Policy CRUD System (Unified) | ⚠️ 70% | 12h | +45 |
| 2 | Agent-Side Policy Application | ⚠️ 60% | 9h | +30 |
| 3 | Agent Filesystem Browser API | ❌ 0% | 11h | +35 |
| 4 | Visual Include/Exclude Rule Builder UI | ❌ 0% | 17h | +40 |
| 5 | Agent Sandbox Configuration & Enforcement | ❌ 0% | 17h | +50 |
| 6 | Repository Credentials & Certificates | ❌ 0% | 19h | +70 |
| 7 | Pre/Post Hook Integration | ⚠️ 30% | 11h | +25 |
| 8 | Policy–Agent Link Management UI | ✅ 90% | 7h | +15 |

**Legend**: ✅ Complete | ⚠️ Partial | ❌ Not Started

---

## 🏗️ Architecture Additions

### New Components

1. **FilesystemService** (Agent)
   - Serves filesystem tree API
   - Enforces sandbox restrictions
   - Handles pagination for large dirs

2. **SandboxEngine** (Agent)
   - Validates paths against whitelist/blacklist
   - Enforces depth limits
   - Parses commands for path references

3. **CredentialStore** (Orchestrator)
   - CRUD for encrypted credentials
   - AES-256-GCM encryption
   - Time-limited access tokens

4. **PolicyValidator** (Orchestrator)
   - Schema validation
   - Conflict detection
   - Background validation jobs

5. **HookIntegrationLayer** (Orchestrator)
   - Fetches hooks from Epic 14
   - Validates hook parameters
   - Attaches hooks to tasks

### Database Changes

**New Table**:
- `credentials` (passwords, certs, AWS/GCS keys)

**Extended Tables**:
- `policies` (+7 columns: sandbox_config, credentials_id, pre_hooks, post_hooks, validation_status, validation_errors, policy_version)
- `agents` (+1 column: sandbox_config)
- `tasks` (+5 columns: credentials_token, pre_hooks, post_hooks, sandbox_config, policy_version)

**Migrations**:
- 007: Policy + Agent extensions
- 008: Credentials table
- 009: Task extensions

---

## 📅 Implementation Phases

### Phase 1: Foundation (Week 1-2) — 40h
- Extend Policy model (sandbox, credentials, hooks)
- Create Credential model
- Implement PolicyValidator
- Conflict detection
- **Deliverable**: US #1 complete, 105 tests

### Phase 2: Security (Week 3-4) — 45h
- SandboxEngine (agent)
- EncryptionService (orchestrator)
- Credential API
- Certificate validation
- **Deliverable**: US #5 + US #6 complete, 110 tests

### Phase 3: Filesystem Browser (Week 5) — 25h
- FilesystemService (agent)
- Orchestrator proxy
- Pagination + caching
- **Deliverable**: US #3 complete, 35 tests

### Phase 4: Visual Rule Builder (Week 6-7) — 40h
- FilesystemTree.vue component
- Pattern generator
- Preview modal
- Conflict highlighting
- **Deliverable**: US #4 complete, 40 tests

### Phase 5: Hook Integration (Week 8) — 15h
- Hook selection UI
- Hook execution in agent
- Hook validation
- **Deliverable**: US #7 + US #2 complete, 25 tests

### Phase 6: Polish (Week 9) — 15h
- Validation status UI
- Bulk assignment
- Background validation job
- Documentation
- **Deliverable**: US #8 complete, 15 tests

---

## 🔗 Dependencies

### Requires (Inbound)
- **Epic 14** (partial): Hook template API
- **Epic 15** ✅ (complete): Concurrency control

### Enables (Outbound)
- **Epic 17**: Preflight validation framework
- **Epic 18**: Smart triggering policy engine
- **Epic 19**: Forecasting analytics
- **Epic 20**: DR/restore credential management
- **Epic 20B**: Policy wizard visual builder

---

## ✅ Success Criteria

### Functional
- [ ] All 8 user stories implemented
- [ ] 310 new tests passing
- [ ] All migrations applied
- [ ] UI components functional
- [ ] API docs updated

### Performance
- [ ] Filesystem tree < 2s (10k files)
- [ ] Policy validation < 500ms
- [ ] Encryption < 100ms
- [ ] Pattern preview < 1s

### Security
- [ ] Credentials encrypted (AES-256-GCM)
- [ ] Tokens expire (1h TTL)
- [ ] Sandbox blocks forbidden paths
- [ ] Certificate expiry detection

### Usability
- [ ] Visual rule builder intuitive
- [ ] Clear conflict error messages
- [ ] Actionable validation warnings
- [ ] Bulk assignment time savings

---

## 🚨 Known Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Epic 14 hooks not ready | Medium | Medium | Defer US #7, implement placeholder |
| Encryption key complexity | High | Low | Use crypto/aes, document rotation |
| Filesystem browser slow | Medium | Medium | Pagination, caching, lazy load |
| Sandbox bypass | Critical | Low | Security testing, audit |

---

## 📦 What's Already Done

### From Previous EPICs

| Feature | Source | Status |
|---------|--------|--------|
| Policy CRUD | Epic 6 | ✅ 100% |
| Policy-Agent Assignment | Epic 7 | ✅ 100% |
| Basic Policy UI | Phase 1-2 | ✅ 100% |
| Repository Config | Policy model | ⚠️ Basic |
| Scheduler | Epic 14 | ✅ 100% |
| Concurrency Control | Epic 15 | ✅ 100% |

**Current Test Count**: 679 tests passing

---

## 🎯 Next Steps

1. **Immediate** (Today)
   - ✅ Review epic16-status.md
   - Get stakeholder approval
   - Confirm 180h timeline

2. **Week 1** (Phase 1 Start)
   - Set up test infrastructure
   - Create epic16-phase1 branch
   - Extend Policy model
   - Write first PolicyValidator tests

3. **Week 9** (Phase 6 End)
   - All 8 user stories complete
   - 989 tests passing
   - Ready for Epic 17

---

## 📖 Documentation

- **Full Spec**: `/docs/epic16-status.md` (comprehensive)
- **Quick Ref**: `/docs/epic16-summary.md` (this file)
- **Architecture**: `/docs/architecture.md` (system design)
- **Backlog**: `/docs/backlog.md` (all epics)

---

## 💡 Key Innovations

1. **Visual Filesystem Browser**: First backup solution with agent-side FS browsing
2. **Sandbox Security**: Prevent accidental sensitive data exposure
3. **Credential Encryption**: Secure storage with time-limited delivery
4. **Conflict Detection**: Catch policy errors before execution
5. **TDD-First**: All 310 tests written before implementation

---

**Last Updated**: 2025-11-26  
**Full Details**: See `/docs/epic16-status.md`
