# Phase 1 - Mocking Infrastructure Recommendations

**Date**: 2025-11-10
**Status**: ⚠️  ARCHITECTURAL RECOMMENDATIONS
**Priority**: HIGH - Blocks 3+ packages from achieving target coverage

---

## 🎯 Problem Statement

**3 packages blocked** from achieving 50%+ coverage due to external dependencies:

| Package | Current Coverage | Target | Blocker |
|---------|------------------|--------|---------|
| internal/task | 28.6% | 50%+ | database.Pool (70% of code) |
| internal/deployment | 15.0% | 50%+ | SSH, security scans, server health |
| internal/cognee | 12.5% | 50%+ | External Cognee API |
| internal/providers | 0.0% | 90%+ | ProviderManager from memory package |

**Impact**: ~25% of internal packages cannot be adequately tested without mocking infrastructure

---

## 🚧 Current Architecture Issues

### 1. Database Dependency - internal/task

**Problem**:
```go
// database/database.go
type Database struct {
    Pool *pgxpool.Pool  // Concrete type, not mockable
}

// task/manager.go
type TaskManager struct {
    db *database.Database  // Direct dependency on concrete type
}
```

**Blocked Functions** (70% of internal/task):
- All checkpoint.go functions (5 functions, 0% coverage)
- Most dependency.go functions (6/9 functions, 0% coverage)
- All manager_db.go functions (7 functions, 0% coverage)

**Root Cause**: No abstraction layer between business logic and database operations

### 2. External System Dependencies - internal/deployment

**Problem**:
- SSH connections to deployment servers
- Security scanning APIs
- Health check endpoints
- No interfaces for external operations

**Blocked Functions** (85% of internal/deployment):
- All deployment execution functions (0% coverage)
- All validation and health check functions (0% coverage)
- All rollback and monitoring functions (0% coverage)

### 3. External API Dependencies - internal/cognee

**Problem**:
- Direct HTTP calls to Cognee API
- No interface abstraction for API client
- Failing tests due to unexpected state

---

## ✅ Recommended Solutions

### Solution 1: Repository Pattern for Database Operations

**Create DatabaseInterface**:
```go
// database/interface.go
package database

type DatabaseInterface interface {
    // Task operations
    CreateTask(ctx context.Context, task *Task) error
    GetTask(ctx context.Context, id string) (*Task, error)
    UpdateTask(ctx context.Context, task *Task) error
    ListTasks(ctx context.Context, filters *TaskFilters) ([]*Task, error)

    // Checkpoint operations
    CreateCheckpoint(ctx context.Context, checkpoint *Checkpoint) error
    GetCheckpoints(ctx context.Context, taskID string) ([]*Checkpoint, error)
    DeleteCheckpoint(ctx context.Context, id string) error

    // Generic query operations
    Exec(ctx context.Context, query string, args ...interface{}) error
    Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
    QueryRow(ctx context.Context, query string, args ...interface{}) Row
}

type Rows interface {
    Next() bool
    Scan(dest ...interface{}) error
    Close() error
}

type Row interface {
    Scan(dest ...interface{}) error
}
```

**Update TaskManager**:
```go
// task/manager.go
type TaskManager struct {
    db database.DatabaseInterface  // Interface, not concrete type
    redis *redis.Client
    logger *logging.Logger
}

func NewTaskManager(db database.DatabaseInterface, redis *redis.Client) *TaskManager {
    return &TaskManager{
        db:     db,
        redis:  redis,
        logger: logging.NewLogger(logging.INFO),
    }
}
```

**Create Mock Implementation**:
```go
// database/mock_database.go
package database

type MockDatabase struct {
    // Use testify/mock for flexible mocking
    mock.Mock

    // Or maintain internal state for simple cases
    tasks       map[string]*Task
    checkpoints map[string][]*Checkpoint
}

func NewMockDatabase() *MockDatabase {
    return &MockDatabase{
        tasks:       make(map[string]*Task),
        checkpoints: make(map[string][]*Checkpoint),
    }
}

func (m *MockDatabase) CreateTask(ctx context.Context, task *Task) error {
    args := m.Called(ctx, task)
    if args.Error(0) != nil {
        return args.Error(0)
    }
    m.tasks[task.ID] = task
    return nil
}

// ... implement all interface methods
```

**Migration Steps**:
1. Create `database/interface.go` with DatabaseInterface
2. Update `database/database.go` to implement interface
3. Create `database/mock_database.go` with mock implementation
4. Update all packages to accept DatabaseInterface instead of *Database
5. Update tests to use MockDatabase

**Estimated Effort**: 2-3 days

### Solution 2: Service Interfaces for External Systems

**Create DeploymentService Interface**:
```go
// deployment/interface.go
package deployment

type DeploymentService interface {
    // Security operations
    RunSecurityScan(ctx context.Context, config *SecurityScanConfig) (*SecurityResult, error)

    // Server operations
    ConnectToServer(ctx context.Context, server *ServerConfig) (ServerConnection, error)
    DeployToServer(ctx context.Context, conn ServerConnection, artifact *Artifact) error

    // Health operations
    CheckServerHealth(ctx context.Context, server *ServerConfig) (*HealthStatus, error)
}

type ServerConnection interface {
    Execute(command string) (string, error)
    Upload(localPath, remotePath string) error
    Close() error
}

// Mock implementation
type MockDeploymentService struct {
    mock.Mock
}

func (m *MockDeploymentService) RunSecurityScan(ctx context.Context, config *SecurityScanConfig) (*SecurityResult, error) {
    args := m.Called(ctx, config)
    if result := args.Get(0); result != nil {
        return result.(*SecurityResult), args.Error(1)
    }
    return nil, args.Error(1)
}
```

**Benefits**:
- Test business logic without actual SSH connections
- Simulate deployment scenarios (success, failure, partial)
- Test error handling and rollback logic

**Estimated Effort**: 1-2 days

### Solution 3: HTTP Client Interface for API Calls

**Create APIClient Interface**:
```go
// cognee/client.go
package cognee

type CogneeClient interface {
    Search(ctx context.Context, query *SearchQuery) (*SearchResult, error)
    Store(ctx context.Context, data *GraphData) error
    Delete(ctx context.Context, id string) error
}

type HTTPCogneeClient struct {
    baseURL string
    apiKey  string
    client  *http.Client
}

func (c *HTTPCogneeClient) Search(ctx context.Context, query *SearchQuery) (*SearchResult, error) {
    // Actual HTTP implementation
}

type MockCogneeClient struct {
    mock.Mock
    data map[string]*GraphData  // In-memory storage for testing
}

func (m *MockCogneeClient) Search(ctx context.Context, query *SearchQuery) (*SearchResult, error) {
    args := m.Called(ctx, query)
    if result := args.Get(0); result != nil {
        return result.(*SearchResult), args.Error(1)
    }
    return nil, args.Error(1)
}
```

**Benefits**:
- Test without external API dependency
- Simulate API errors and edge cases
- Faster test execution

**Estimated Effort**: 1 day

---

## 📋 Implementation Checklist

### Phase 1: Database Mocking (Highest Priority)
- [ ] Create `database/interface.go` with DatabaseInterface
- [ ] Update `database/database.go` to implement interface
- [ ] Create `database/mock_database.go` with comprehensive mocks
- [ ] Update `internal/task` to use DatabaseInterface
- [ ] Add tests for task operations using MockDatabase
- [ ] Update `internal/auth` to use DatabaseInterface (if applicable)
- [ ] Update `internal/project` to use DatabaseInterface (if applicable)

### Phase 2: Deployment Service Mocking
- [ ] Create `deployment/interface.go` with service interfaces
- [ ] Create `deployment/mock_service.go` with mocks
- [ ] Refactor deployment logic to use interfaces
- [ ] Add tests for deployment scenarios

### Phase 3: API Client Mocking
- [ ] Create `cognee/client_interface.go`
- [ ] Create `cognee/mock_client.go`
- [ ] Refactor cognee integration to use interface
- [ ] Add comprehensive tests
- [ ] Fix failing test (GetMetrics memory issue)

### Phase 4: Provider Manager Mocking
- [ ] Create interface for ProviderManager
- [ ] Mock ProviderManager for testing
- [ ] Add tests for internal/providers

---

## 📈 Expected Coverage Improvements

### With Database Mocking:
- internal/task: 28.6% → **70%+**
- internal/auth: Current → **+20%**
- internal/project: Current → **+15%**

### With Deployment Mocking:
- internal/deployment: 15.0% → **60%+**

### With API Client Mocking:
- internal/cognee: 12.5% → **60%+**

### With Provider Manager Mocking:
- internal/providers: 0.0% → **80%+**

### Total Impact:
- **4 packages** from low/blocked → 60-80% coverage
- **Estimated total improvement**: +200% across blocked packages

---

## 🎯 Alternative Approaches

### Option A: Integration Tests (Current Limitation)
**Pros**:
- Tests real system behavior
- Catches integration issues

**Cons**:
- Requires running PostgreSQL for tests
- Slow test execution
- Difficult CI/CD setup
- Cannot test error scenarios easily

**Verdict**: Not suitable for unit test coverage goals

### Option B: Test Databases (sqlite, in-memory)
**Pros**:
- Faster than real PostgreSQL
- No external dependencies in tests

**Cons**:
- SQL dialect differences
- Still requires database setup
- Doesn't test pgx-specific features
- Complex schema migrations

**Verdict**: Better than integration tests, but mocking is cleaner

### Option C: Interface Abstraction (Recommended)
**Pros**:
- Clean separation of concerns
- Easy to test
- Fast test execution
- Can simulate any scenario

**Cons**:
- Requires refactoring effort
- Initial implementation time

**Verdict**: ✅ Best long-term solution

---

## 💡 Design Principles for Mocking

### 1. **Dependency Inversion Principle**
```go
// Bad: Depend on concrete type
func NewTaskManager(db *database.Database) *TaskManager

// Good: Depend on interface
func NewTaskManager(db database.DatabaseInterface) *TaskManager
```

### 2. **Interface Segregation**
```go
// Bad: One large interface
type DatabaseInterface interface {
    // 50 methods...
}

// Good: Multiple focused interfaces
type TaskRepository interface {
    CreateTask(ctx context.Context, task *Task) error
    GetTask(ctx context.Context, id string) (*Task, error)
}

type CheckpointRepository interface {
    CreateCheckpoint(ctx context.Context, checkpoint *Checkpoint) error
    GetCheckpoints(ctx context.Context, taskID string) ([]*Checkpoint, error)
}
```

### 3. **Test Data Builders**
```go
// test_helpers.go
func NewTestTask(opts ...TaskOption) *Task {
    task := &Task{
        ID:         uuid.New(),
        Type:       TaskTypePlanning,
        Status:     StatusPending,
        CreatedAt:  time.Now(),
    }

    for _, opt := range opts {
        opt(task)
    }

    return task
}

func WithStatus(status TaskStatus) TaskOption {
    return func(t *Task) {
        t.Status = status
    }
}

// Usage in tests
task := NewTestTask(WithStatus(StatusRunning))
```

---

## 📝 Code Examples

### Example: Testing Task Creation with Mock Database

**Before (not testable)**:
```go
func TestTaskManager_CreateTask(t *testing.T) {
    // Can't test without real database
    t.Skip("Requires database - skipping")
}
```

**After (fully testable)**:
```go
func TestTaskManager_CreateTask(t *testing.T) {
    mockDB := database.NewMockDatabase()
    manager := task.NewTaskManager(mockDB, nil)

    testTask := &task.Task{
        ID:   uuid.New(),
        Type: task.TaskTypePlanning,
    }

    // Set expectations
    mockDB.On("CreateTask", mock.Anything, testTask).Return(nil)

    // Execute
    err := manager.CreateTask(context.Background(), testTask)

    // Assert
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }
    mockDB.AssertExpectations(t)
}

func TestTaskManager_CreateTask_DatabaseError(t *testing.T) {
    mockDB := database.NewMockDatabase()
    manager := task.NewTaskManager(mockDB, nil)

    // Simulate database error
    mockDB.On("CreateTask", mock.Anything, mock.Anything).
        Return(errors.New("database connection lost"))

    err := manager.CreateTask(context.Background(), &task.Task{})

    if err == nil {
        t.Error("Expected error, got nil")
    }
}
```

---

## 🔧 Recommended Tools

### Mocking Libraries:
1. **testify/mock** (already in use)
   - Flexible method expectations
   - Argument matching
   - Call count verification

2. **gomock** (alternative)
   - Code generation
   - Strict type safety
   - IDE support

### Database Testing:
1. **pgxmock** - Mock for pgx specifically
2. **sqlmock** - Generic SQL mocking
3. **testcontainers-go** - Real PostgreSQL in Docker (for integration tests)

### HTTP Mocking:
1. **httptest** - Standard library HTTP testing
2. **gock** - HTTP mock library
3. **httpmock** - Response mocking

---

## 📊 Return on Investment

### Time Investment:
- **Database Interface**: 2-3 days
- **Deployment Service**: 1-2 days
- **API Client Interface**: 1 day
- **Total**: ~5 days of development

### Coverage Improvement:
- **Before**: 3 packages at 12-28% coverage
- **After**: 3 packages at 60-80% coverage
- **Total Improvement**: +150% average across blocked packages

### Long-term Benefits:
- ✅ Faster test execution (no database setup)
- ✅ Easier to test edge cases and errors
- ✅ Better code organization (SOLID principles)
- ✅ Easier onboarding for new developers
- ✅ CI/CD pipeline improvement (no external dependencies)

---

## 🚀 Next Steps

### Immediate (This Session):
1. ✅ Document mocking recommendations (this file)
2. ✅ Create Phase 1 Session 6 Extended summary
3. ⏳ Update IMPLEMENTATION_LOG.txt

### Short-term (Next 1-2 Weeks):
1. Implement database interface and mocks
2. Refactor internal/task to use interface
3. Add comprehensive tests for internal/task
4. Measure coverage improvement

### Medium-term (Next Month):
1. Implement deployment service interfaces
2. Implement API client interfaces
3. Refactor all blocked packages
4. Achieve 60%+ coverage on all blocked packages

### Long-term (Ongoing):
1. Apply interface pattern to all new code
2. Document mocking patterns in CONTRIBUTING.md
3. Create test helper utilities package
4. Establish testing best practices

---

**Document Status**: ✅ COMPLETE
**Next Review**: After database interface implementation
**Owner**: Development Team
**Priority**: HIGH

---

*Documentation created: 2025-11-10*
*Based on Phase 1 Sessions 4-6 findings*
