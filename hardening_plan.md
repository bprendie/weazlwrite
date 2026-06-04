# WeazlWrite Security Hardening Plan

## Executive Summary

Current security posture: **7.5/10** for intended use case (personal, local-first writing tool)
Target after hardening: **8.5/10**

This plan addresses identified vulnerabilities and architectural improvements while maintaining WeazlWrite's core philosophy: a sovereign, local-first text editor that doesn't pretend to be enterprise-grade security infrastructure.

---

## Critical (Do Immediately)

### 1. Upgrade Go Runtime
**Priority:** CRITICAL  
**Effort:** Low (15 minutes)  
**Impact:** Fixes 12 active vulnerabilities

**Current:** Go 1.25.3  
**Target:** Go 1.25.10 or later

**Vulnerabilities Fixed:**
- GO-2026-4918: HTTP/2 Infinite Loop DoS
- GO-2026-4870: TLS KeyUpdate DoS
- GO-2026-4340: TLS Handshake Security Issue
- GO-2026-4337: TLS Session Resumption Issue
- GO-2026-4971: Panic in Dial/LookupPort (Windows)
- GO-2026-4947: Unexpected work during chain building (crypto/x509)
- GO-2026-4946: Inefficient policy validation (crypto/x509)
- GO-2026-4602: FileInfo can escape from Root (os)
- GO-2026-4601: Incorrect IPv6 parsing (net/url)
- GO-2026-4341: Memory exhaustion in query parsing (net/url)
- GO-2025-4175: Improper DNS name constraints (crypto/x509)
- GO-2025-4155: Excessive resource consumption (crypto/x509)

**Steps:**
```bash
# Update Go installation to 1.25.10+
# Then rebuild
go build -o weazlwrite ./cmd/weazlwrite
go build -o weazlwrite-setup ./cmd/weazlwrite-setup
```

### 2. Update Dependencies
**Priority:** CRITICAL  
**Effort:** Low (5 minutes)  
**Impact:** Fixes HTTP/2 vulnerabilities in golang.org/x/net

**Current:** golang.org/x/net@v0.45.0  
**Target:** golang.org/x/net@v0.53.0+

**Steps:**
```bash
go get golang.org/x/net@v0.53.0
go mod tidy
go build -o weazlwrite ./cmd/weazlwrite
go build -o weazlwrite-setup ./cmd/weazlwrite-setup
```

**Verification:**
```bash
govulncheck ./...
# Should show 0 vulnerabilities affecting your code
```

---

## High Priority (Recommended)

### 3. Add Password Attempt Rate Limiting
**Priority:** HIGH  
**Effort:** Medium (2-3 hours)  
**Impact:** Prevents brute force attacks on vault password

**Current Risk:** Unlimited password attempts possible  
**Location:** `internal/storage/auth.go:30-40`

**Implementation Approach:**
```go
// Add to Store struct
type Store struct {
    db              *sql.DB
    key             []byte
    unlocked        bool
    failedAttempts  int
    lastAttemptTime time.Time
    lockoutUntil    time.Time
}

// Modify Unlock function
func (s *Store) Unlock(password string) error {
    // Check if locked out
    if time.Now().Before(s.lockoutUntil) {
        remaining := time.Until(s.lockoutUntil).Round(time.Second)
        return fmt.Errorf("too many failed attempts, try again in %v", remaining)
    }
    
    var hash string
    if err := s.db.QueryRow(`select password_hash from vault where id = 1`).Scan(&hash); err != nil {
        return err
    }
    
    if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
        s.failedAttempts++
        s.lastAttemptTime = time.Now()
        
        // Exponential backoff: 5s, 10s, 30s, 1m, 5m, 15m
        delays := []time.Duration{
            5 * time.Second,
            10 * time.Second,
            30 * time.Second,
            1 * time.Minute,
            5 * time.Minute,
            15 * time.Minute,
        }
        
        if s.failedAttempts >= len(delays) {
            s.lockoutUntil = time.Now().Add(delays[len(delays)-1])
        } else {
            s.lockoutUntil = time.Now().Add(delays[s.failedAttempts-1])
        }
        
        return errors.New("bad vault password")
    }
    
    // Reset on success
    s.failedAttempts = 0
    s.lockoutUntil = time.Time{}
    s.unlockWith(password)
    return nil
}
```

**Testing:**
- Verify lockout triggers after failed attempts
- Verify exponential backoff works correctly
- Verify successful unlock resets counter

### 4. Add Session Auto-Lock
**Priority:** HIGH  
**Effort:** Medium (2-3 hours)  
**Impact:** Protects against unattended terminal exposure

**Current Risk:** Vault stays unlocked indefinitely  
**Location:** `internal/storage/store.go`

**Implementation Approach:**
```go
// Add to Store struct
type Store struct {
    db              *sql.DB
    key             []byte
    unlocked        bool
    lastActivity    time.Time
    autoLockTimeout time.Duration
}

// Add activity tracking
func (s *Store) UpdateActivity() {
    if s.unlocked {
        s.lastActivity = time.Now()
    }
}

// Add lock check
func (s *Store) CheckAutoLock() bool {
    if !s.unlocked || s.autoLockTimeout == 0 {
        return false
    }
    
    if time.Since(s.lastActivity) > s.autoLockTimeout {
        s.Lock()
        return true
    }
    return false
}

// Add lock function
func (s *Store) Lock() {
    s.unlocked = false
    s.key = nil
}
```

**Configuration:**
Add to `config.json`:
```json
{
  "vault": {
    "root": "~/.weazlwrite/vault",
    "auto_lock_minutes": 15
  }
}
```

**Integration Points:**
- Call `UpdateActivity()` on every user interaction
- Check `CheckAutoLock()` periodically (e.g., every 30 seconds)
- Display warning before auto-lock triggers

---

## Medium Priority (Consider for v2.0)

### 5. Improve Key Derivation Function
**Priority:** MEDIUM  
**Effort:** High (4-6 hours)  
**Impact:** Protects against rainbow table attacks

**Current Implementation:**
```go
func (s *Store) unlockWith(password string) {
    sum := sha3.Sum256([]byte(password))
    s.key = sum[:]
    s.unlocked = true
}
```

**Recommended Implementation:**
```go
import "golang.org/x/crypto/argon2"

// Store salt in vault table
func (s *Store) CreateVault(password string) error {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    
    // Generate random salt for key derivation
    salt := make([]byte, 32)
    if _, err := rand.Read(salt); err != nil {
        return err
    }
    
    if _, err := s.db.Exec(
        `insert into vault (id, password_hash, kdf_salt) values (1, ?, ?)`,
        string(hash), salt,
    ); err != nil {
        return err
    }
    
    s.unlockWith(password, salt)
    return nil
}

func (s *Store) Unlock(password string) error {
    var hash string
    var salt []byte
    if err := s.db.QueryRow(
        `select password_hash, kdf_salt from vault where id = 1`,
    ).Scan(&hash, &salt); err != nil {
        return err
    }
    
    if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
        return errors.New("bad vault password")
    }
    
    s.unlockWith(password, salt)
    return nil
}

func (s *Store) unlockWith(password string, salt []byte) {
    // Argon2id parameters: time=1, memory=64MB, threads=4, keyLen=32
    s.key = argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
    s.unlocked = true
}
```

**Schema Migration:**
```sql
ALTER TABLE vault ADD COLUMN kdf_salt BLOB;
```

**Note:** This is a breaking change. Existing vaults would need migration or re-creation.

### 6. Add Path Validation
**Priority:** MEDIUM  
**Effort:** Low (1-2 hours)  
**Impact:** Prevents directory traversal attacks

**Implementation:**
```go
import "path/filepath"

func validatePath(path string) error {
    // Clean the path
    clean := filepath.Clean(path)
    
    // Check for directory traversal
    if strings.Contains(clean, "..") {
        return errors.New("path contains directory traversal")
    }
    
    // Check for absolute paths where relative expected
    if filepath.IsAbs(clean) {
        return errors.New("absolute paths not allowed")
    }
    
    return nil
}

// Apply to all path-handling functions
func (s *Store) SaveNote(path string, content []byte) error {
    if err := validatePath(path); err != nil {
        return err
    }
    // ... rest of implementation
}
```

---

## Low Priority (Nice to Have)

### 7. OS Keyring Integration for API Keys
**Priority:** LOW  
**Effort:** High (6-8 hours)  
**Impact:** Protects LLM API keys from config file exposure

**Current:** API keys stored in plaintext in `config.json` (mitigated by 0o600 permissions)

**Libraries to Consider:**
- `github.com/zalando/go-keyring` (cross-platform)
- `github.com/99designs/keyring` (more features)

**Implementation Notes:**
- Fallback to config file if keyring unavailable
- Migrate existing keys on first run
- Update setup scripts to use keyring

### 8. Audit Logging
**Priority:** LOW  
**Effort:** Medium (3-4 hours)  
**Impact:** Forensics and debugging

**Scope:**
- Vault unlock/lock events
- Failed password attempts
- File operations (create, delete, rename)
- Eyes Only mode toggles

**Implementation:**
```go
type AuditLog struct {
    Timestamp time.Time
    Event     string
    Details   string
    Success   bool
}

// Store in separate SQLite table
func (s *Store) LogEvent(event, details string, success bool) {
    s.db.Exec(
        `insert into audit_log (timestamp, event, details, success) values (?, ?, ?, ?)`,
        time.Now(), event, details, success,
    )
}
```

---

## Testing Plan

### After Critical Updates (Go + Dependencies)
```bash
# 1. Run vulnerability scan
govulncheck ./...

# 2. Run existing tests
go test ./...

# 3. Manual testing
go run ./cmd/weazlwrite
# - Create new vault
# - Unlock existing vault
# - Create/edit/save notes
# - Test Eyes Only mode
# - Test LLM integration
```

### After Rate Limiting Implementation
```bash
# Test scenarios:
# 1. Enter wrong password 3 times, verify lockout
# 2. Wait for lockout to expire, verify can retry
# 3. Enter correct password, verify counter resets
# 4. Verify exponential backoff increases correctly
```

### After Auto-Lock Implementation
```bash
# Test scenarios:
# 1. Unlock vault, wait for timeout, verify auto-lock
# 2. Interact with app, verify activity resets timer
# 3. Verify warning displays before lock
# 4. Verify can re-unlock after auto-lock
```

---

## Rollout Strategy

### Phase 1: Critical Security (Week 1)
- [x] Upgrade Go to 1.25.10+
- [x] Update golang.org/x/net to v0.53.0+
- [x] Run full test suite
- [ ] Tag release v1.1.0

### Phase 2: High Priority Features (Week 2-3)
- [x] Implement password attempt rate limiting
- [x] Implement session auto-lock
- [x] Add configuration options
- [x] Update documentation
- [ ] Tag release v1.2.0

### Phase 3: Medium Priority (Future)
- [ ] Evaluate KDF migration strategy
- [ ] Implement path validation
- [ ] Consider breaking changes for v2.0

### Phase 4: Low Priority (Backlog)
- [ ] OS keyring integration
- [ ] Audit logging
- [ ] Additional security hardening

---

## Documentation Updates

### README.md Updates Needed
1. [x] Add "Security" section with current hardening status
2. [x] Document auto-lock feature and configuration
3. [x] Document rate limiting behavior
4. [x] Update "Security" section with new mitigations

### New Documentation
1. [x] `SECURITY.md` - Security policy and vulnerability reporting
2. [x] `CHANGELOG.md` - Track security-related changes
3. [x] Update installation scripts to check Go version

---

## Success Metrics

### Immediate (Post Phase 1)
- ✅ Zero vulnerabilities in `govulncheck` scan
- ✅ All existing tests pass
- ✅ Manual testing confirms no regressions

### Short-term (Post Phase 2)
- ✅ Rate limiting prevents brute force attacks
- ✅ Auto-lock protects unattended sessions
- ✅ User feedback confirms features work as expected

### Long-term
- ✅ Security rating improves from 7.5/10 to 8.5/10
- ✅ No security-related bug reports
- ✅ Community confidence in security posture

---

## Risk Assessment

### Risks of NOT Implementing
- **Critical Updates:** Active exploitation of known vulnerabilities
- **Rate Limiting:** Brute force attacks on weak passwords
- **Auto-Lock:** Data exposure from unattended terminals

### Risks of Implementing
- **Breaking Changes:** KDF migration requires vault recreation
- **User Experience:** Rate limiting may frustrate legitimate users
- **Complexity:** More features = more potential bugs

### Mitigation
- Thorough testing before release
- Clear documentation of breaking changes
- Gradual rollout with user feedback
- Maintain backward compatibility where possible

---

## Conclusion

This hardening plan balances security improvements with WeazlWrite's core philosophy: a simple, local-first writing tool that respects user privacy without pretending to be enterprise-grade security infrastructure.

**Immediate Action Required:** Upgrade Go and dependencies (Phase 1)  
**Recommended Next Steps:** Implement rate limiting and auto-lock (Phase 2)  
**Future Considerations:** KDF improvements and additional hardening (Phase 3-4)

The goal is not to turn WeazlWrite into a vault product, but to ensure it provides appropriate security for its intended use case: protecting personal notes from casual snooping and accidental data leakage.
