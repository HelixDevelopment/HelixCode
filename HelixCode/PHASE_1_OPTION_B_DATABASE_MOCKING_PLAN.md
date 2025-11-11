# Phase 1 - Option B: Database Mocking Infrastructure

**Created**: 2025-11-11
**Status**: 📋 PLANNING → READY TO IMPLEMENT
**Estimated Duration**: 3-5 days
**Expected ROI**: +200% coverage across 3 packages

---

## 🎯 Mission

Implement a comprehensive database mocking infrastructure to unblock test coverage for 3 major packages that are currently blocked by direct database dependencies.

---

## 📊 Current State Analysis

### Blocked Packages

| Package | Coverage | Primary Blocker | Functions Blocked | Potential Gain |
|---------|----------|-----------------|-------------------|----------------|
| internal/task | 28.6% | `*database.Database` (Pool) | ~70% of code | +40-50% |
| internal/auth | 47.0% | `*database.Database` (Pool) | auth_db.go (~53%) | +20-30% |
| internal/project | 32.8% | `*database.Database` (Pool) | ~50-60% | +30-40% |

**Total Potential Gain**: +90-120 percentage points across 3 packages

### Database Usage Patterns Identified

From `internal/task` analysis, the code uses these `pgxpool.Pool` methods:

1. **`Exec(ctx, query, args...)`**
   - Used for: INSERT, UPDATE, DELETE operations
   - Returns: CommandTag (rows affected count)
   - Examples: CreateTask, UpdateTask, DeleteTask, SetCheckpoint

2. **`Query(ctx, query, args...)`**
   - Used for: SELECT operations returning multiple rows
   - Returns: Rows (iterator)
   - Examples: ListTasks, GetCheckpoints, GetDependencies

3. **`QueryRow(ctx, query, args...)`**
   - Used for: SELECT operations returning single row
   - Returns: Row (single result)
   - Examples: GetTask, GetTaskByID, CheckDependency

4. **`Ping(ctx)`**
   - Used for: Health checks
   - Returns: error

---

## 🏗️ Architecture Solution

### Option 1: Wrap pgxpool.Pool (RECOMMENDED) ✅

Create a thin interface wrapper around `pgxpool.Pool` methods:

**Pros**:
- Minimal changes to existing code
- Direct mapping to pgx methods
- Easy to mock with testify/mock
- Preserves pgx-specific features
- Simple interface

**Cons**:
- Tightly coupled to pgx API
- Cannot easily switch database drivers

### Option 2: Generic DatabaseInterface

Create a generic database interface independent of pgx:

**Pros**:
- Database-agnostic
- Can swap drivers easily
- Clean abstraction

**Cons**:
- More abstraction layers
- More code changes required
- May lose pgx-specific features
- Higher complexity

**Decision**: Use **Option 1** for pragmatic reasons - minimal code changes, maintains pgx benefits, sufficient for testing needs.

---

## 📋 Implementation Plan

### Phase 1: Interface Definition (Day 1 - 4 hours)

#### Step 1.1: Create Database Interface

**File**: `internal/database/interface.go`

```go
package database

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DatabaseInterface defines the database operations interface
// This allows mocking for testing while maintaining pgx semantics
type DatabaseInterface interface {
	// Query execution
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row

	// Connection management
	Ping(ctx context.Context) error
	Close()

	// Optional: Advanced features (can be added later)
	// Begin(ctx context.Context) (pgx.Tx, error)
	// CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

// Ensure Database implements DatabaseInterface
var _ DatabaseInterface = (*Database)(nil)
```

**Key Design Decisions**:
- Use `pgx.Rows` and `pgx.Row` interfaces (already interfaces!)
- Use `pgconn.CommandTag` for Exec results
- Keep pgx context patterns
- Add methods as needed (YAGNI principle)

#### Step 1.2: Implement Interface on Existing Database

**File**: `internal/database/database.go` (modifications)

```go
// Exec executes a query without returning any rows
func (db *Database) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	return db.Pool.Exec(ctx, sql, arguments...)
}

// Query executes a query that returns rows
func (db *Database) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.Pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns at most one row
func (db *Database) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.Pool.QueryRow(ctx, sql, args...)
}

// Ping verifies the database connection
func (db *Database) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Close is already implemented
```

---

### Phase 2: Mock Implementation (Day 1-2 - 6 hours)

#### Step 2.1: Create MockDatabase

**File**: `internal/database/mock_database.go`

```go
package database

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
)

// MockDatabase is a mock implementation of DatabaseInterface for testing
type MockDatabase struct {
	mock.Mock
}

// NewMockDatabase creates a new mock database
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{}
}

// Exec mocks the Exec method
func (m *MockDatabase) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

// Query mocks the Query method
func (m *MockDatabase) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	args := m.Called(ctx, sql, arguments)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(pgx.Rows), args.Error(1)
}

// QueryRow mocks the QueryRow method
func (m *MockDatabase) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgx.Row)
}

// Ping mocks the Ping method
func (m *MockDatabase) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Close mocks the Close method
func (m *MockDatabase) Close() {
	m.Called()
}

// Ensure MockDatabase implements DatabaseInterface
var _ DatabaseInterface = (*MockDatabase)(nil)
```

#### Step 2.2: Create Test Helpers

**File**: `internal/database/mock_helpers.go`

```go
package database

import (
	"context"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
)

// MockExecSuccess sets up a successful Exec expectation
func (m *MockDatabase) MockExecSuccess(rowsAffected int64) *mock.Call {
	tag := pgconn.NewCommandTag("INSERT 0 " + fmt.Sprint(rowsAffected))
	return m.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(tag, nil)
}

// MockExecError sets up a failed Exec expectation
func (m *MockDatabase) MockExecError(err error) *mock.Call {
	tag := pgconn.CommandTag{}
	return m.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(tag, err)
}

// MockQueryRowSuccess sets up a successful QueryRow expectation
func (m *MockDatabase) MockQueryRowSuccess(row pgx.Row) *mock.Call {
	return m.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(row)
}

// Additional helpers as needed...
```

---

### Phase 3: Refactor Blocked Packages (Day 2-4 - 12 hours)

#### Step 3.1: Refactor internal/task (4 hours)

**Files to modify**:
- `internal/task/manager.go`
- `internal/task/manager_db.go`
- `internal/task/checkpoint.go`
- `internal/task/dependency.go`

**Changes**:
```go
// Before
type TaskManager struct {
	db *database.Database
	// ...
}

// After
type TaskManager struct {
	db database.DatabaseInterface  // Use interface instead of concrete type
	// ...
}

// Constructor change
func NewTaskManager(db database.DatabaseInterface, ...) *TaskManager {
	return &TaskManager{
		db: db,
		// ...
	}
}

// All db.Pool.Exec() calls become db.Exec()
// All db.Pool.Query() calls become db.Query()
// All db.Pool.QueryRow() calls become db.QueryRow()
```

**Effort**: 30 occurrences to update (~1 hour for changes, 3 hours for testing)

#### Step 3.2: Refactor internal/auth (2-3 hours)

**Files to modify**:
- `internal/auth/service.go`
- `internal/auth/auth_db.go`

**Note**: internal/auth ALREADY has `AuthRepository` interface! This is excellent architecture. We just need to:
1. Ensure it uses the new DatabaseInterface
2. Create MockAuthRepository if not exists
3. Add tests for auth_db.go

**Effort**: Lower than task because better architecture already exists

#### Step 3.3: Refactor internal/project (4 hours)

**Files to modify**:
- `internal/project/manager.go`
- `internal/project/storage.go` (if exists)

**Similar pattern to task package**

---

### Phase 4: Add Tests (Day 4-5 - 8 hours)

#### Step 4.1: Add Tests for internal/task

**New file**: `internal/task/manager_db_test.go`

**Test Examples**:
```go
func TestTaskManager_CreateTask(t *testing.T) {
	mockDB := database.NewMockDatabase()
	manager := NewTaskManager(mockDB, nil)

	testTask := &Task{
		ID:   "test-123",
		Type: TaskTypePlanning,
		// ...
	}

	// Setup expectations
	mockDB.MockExecSuccess(1) // 1 row inserted

	// Execute
	err := manager.CreateTask(context.Background(), testTask)

	// Verify
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestTaskManager_GetTask(t *testing.T) {
	mockDB := database.NewMockDatabase()
	manager := NewTaskManager(mockDB, nil)

	// Create mock row with task data
	mockRow := createMockTaskRow("task-123", TaskTypeBuilding, StatusPending)
	mockDB.MockQueryRowSuccess(mockRow)

	// Execute
	task, err := manager.GetTask(context.Background(), "task-123")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "task-123", task.ID)
	mockDB.AssertExpectations(t)
}
```

**Estimated Coverage Gain**: internal/task: 28.6% → 70%+ (+40%)

#### Step 4.2: Add Tests for internal/auth (auth_db.go)

**Estimated Coverage Gain**: internal/auth: 47.0% → 75%+ (+28%)

#### Step 4.3: Add Tests for internal/project

**Estimated Coverage Gain**: internal/project: 32.8% → 70%+ (+37%)

---

## 📊 Expected Results

### Coverage Improvements

| Package | Before | After (Target) | Gain |
|---------|--------|----------------|------|
| internal/task | 28.6% | 70%+ | +41.4% |
| internal/auth | 47.0% | 75%+ | +28.0% |
| internal/project | 32.8% | 70%+ | +37.2% |
| **Total** | **3 packages** | **Combined** | **+106.6%** |

### Code Metrics

**New Code**:
- `interface.go`: ~40 lines
- Interface methods on Database: ~30 lines
- `mock_database.go`: ~80 lines
- `mock_helpers.go`: ~60 lines
- **Total Infrastructure**: ~210 lines

**Modified Code**:
- internal/task: ~30 occurrences (change db.Pool to db)
- internal/auth: ~15 occurrences
- internal/project: ~25 occurrences
- **Total Changes**: ~70 occurrences

**New Tests**:
- internal/task tests: ~800 lines
- internal/auth tests: ~400 lines
- internal/project tests: ~600 lines
- **Total New Tests**: ~1,800 lines

---

## 🗓️ Implementation Schedule

### Day 1: Foundation (8 hours)
- ✅ Planning complete (this document)
- [ ] Create DatabaseInterface (1 hour)
- [ ] Implement interface on Database (1 hour)
- [ ] Create MockDatabase (2 hours)
- [ ] Create test helpers (2 hours)
- [ ] Write interface tests (2 hours)

### Day 2: Refactor Task Package (8 hours)
- [ ] Update TaskManager to use interface (2 hours)
- [ ] Update all db.Pool calls in task package (2 hours)
- [ ] Verify compilation (1 hour)
- [ ] Start adding tests for task package (3 hours)

### Day 3: Complete Task + Start Auth (8 hours)
- [ ] Complete task package tests (4 hours)
- [ ] Verify task package coverage improvement (30 min)
- [ ] Refactor auth package to use interface (2 hours)
- [ ] Start auth package tests (1.5 hours)

### Day 4: Complete Auth + Start Project (8 hours)
- [ ] Complete auth package tests (3 hours)
- [ ] Verify auth coverage improvement (30 min)
- [ ] Refactor project package to use interface (2 hours)
- [ ] Start project package tests (2.5 hours)

### Day 5: Complete + Validate (6-8 hours)
- [ ] Complete project package tests (3 hours)
- [ ] Verify project coverage improvement (30 min)
- [ ] Run full test suite (30 min)
- [ ] Create documentation (2 hours)
- [ ] Update PHASE_1_MASTER_PROGRESS.md (1 hour)

---

## ✅ Success Criteria

### Technical Criteria
- [ ] DatabaseInterface created and documented
- [ ] MockDatabase fully implements interface
- [ ] All 3 packages refactored to use interface
- [ ] All existing tests still pass
- [ ] internal/task coverage: 28.6% → 60%+ (target: 70%)
- [ ] internal/auth coverage: 47.0% → 65%+ (target: 75%)
- [ ] internal/project coverage: 32.8% → 60%+ (target: 70%)

### Quality Criteria
- [ ] No breaking changes to existing code
- [ ] Clean interface design (SOLID principles)
- [ ] Comprehensive mock helper functions
- [ ] Well-documented patterns for future use
- [ ] All tests passing with no flakiness

### Documentation Criteria
- [ ] Interface design documented
- [ ] Mock usage examples provided
- [ ] Testing patterns documented
- [ ] CONTRIBUTING.md updated with patterns
- [ ] Session summaries for each day

---

## 🚨 Potential Risks & Mitigations

### Risk 1: pgx-specific types hard to mock
**Mitigation**: Use pgx's own interface types (pgx.Rows, pgx.Row) which are already mockable

### Risk 2: Extensive code changes break things
**Mitigation**:
- Make changes incrementally package by package
- Run tests after each package
- Keep git commits atomic

### Risk 3: Mock complexity for complex queries
**Mitigation**:
- Create helper functions for common patterns
- Start with simple test cases
- Build up complexity gradually

### Risk 4: Time estimation too optimistic
**Mitigation**:
- Built in buffer (3-5 days = 3 is optimistic, 5 is realistic)
- Can pause and resume between packages
- Already have clear success criteria per package

---

## 📚 Resources & References

### Go Mocking Libraries
- **testify/mock**: https://pkg.go.dev/github.com/stretchr/testify/mock
- **pgx interfaces**: https://pkg.go.dev/github.com/jackc/pgx/v5

### Existing Patterns in Codebase
- **internal/auth**: Already has AuthRepository interface ✅
- **Pattern to follow**: Use this as reference

### Documentation to Create
- **CONTRIBUTING.md addition**: "Testing with Database Mocks"
- **Database interface design decisions**
- **Mock usage examples**

---

## 💡 Key Insights

### Why This Will Succeed

1. **Clear Blocker**: Database dependency is the main blocker for 3 packages
2. **High ROI**: +100% coverage across 3 packages for ~40 hours work
3. **Proven Pattern**: internal/auth already uses this pattern successfully
4. **Pragmatic Design**: Thin wrapper, minimal abstraction
5. **Testify Support**: Using industry-standard mocking library

### What Makes This Different from Quick Wins

**Quick Wins** (Sessions 8-9):
- Pure logic functions
- No dependencies
- 1 hour per session
- +2-3% per session

**Database Mocking** (Option B):
- Infrastructure change
- Unblocks major packages
- 3-5 days investment
- +100% combined coverage

**Analogy**: Quick wins are "picking low-hanging fruit", Option B is "building a ladder to reach the rest"

---

## 🎯 Next Steps

### Immediate (Start Now!)

1. **Read and approve this plan**
2. **Create Day 1 todos in Phase 1 tracker**
3. **Begin implementation** with interface.go
4. **Track progress daily** in implementation log

### After Completion

1. **Document patterns** in CONTRIBUTING.md
2. **Update NEXT_STEPS.md** with new priorities
3. **Plan Phase 2**: What to tackle next?
4. **Celebrate**: Major milestone in Phase 1! 🎉

---

**Status**: 📋 READY TO IMPLEMENT
**Confidence**: HIGH (95%)
**Expected Start**: 2025-11-11 (After Sessions 8 & 9)
**Expected Completion**: 2025-11-15 (5 business days)

---

*Plan created: 2025-11-11*
*Last updated: 2025-11-11*
*Ready for implementation: YES ✅*
