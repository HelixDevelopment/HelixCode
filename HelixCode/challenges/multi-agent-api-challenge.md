# HelixCode Multi-Agent API Challenge

## Challenge Overview

Create a distributed multi-agent system using HelixCode's API endpoints that demonstrates intelligent task division, coordination, and execution across different development workflows.

## Challenge Requirements

### Core Components
1. **Multi-Agent Coordination System**
   - Create at least 3 different agent types with specialized capabilities
   - Implement inter-agent communication and task handoff
   - Demonstrate intelligent task assignment based on agent capabilities

2. **Distributed Task Execution**
   - Use HelixCode's task management system to distribute work
   - Implement task dependencies and checkpointing
   - Show work preservation across system restarts

3. **Development Workflow Integration**
   - Execute at least 2 different development workflows (planning, building, testing, or refactoring)
   - Demonstrate workflow step dependencies and execution order
   - Show real-time progress tracking

### Technical Requirements

#### API Integration
- Use HelixCode's REST API endpoints for:
  - User authentication and session management
  - Project creation and management
  - Task creation, assignment, and tracking
  - Worker management and health monitoring
  - Workflow execution

#### Multi-Agent Architecture
- **Planning Agent**: Analyzes requirements and creates task breakdown
- **Building Agent**: Handles code generation and integration
- **Testing Agent**: Executes tests and validates results
- **Coordination Agent**: Manages inter-agent communication and task assignment

#### Data Persistence
- Use HelixCode's PostgreSQL database for state persistence
- Implement proper error handling and rollback mechanisms
- Demonstrate checkpointing for long-running tasks

## Success Criteria

### Functional Requirements
1. ✅ System can create and authenticate users via `/api/v1/auth/register` and `/api/v1/auth/login`
2. ✅ System can create projects via `/api/v1/projects` endpoint
3. ✅ System can create and manage tasks via `/api/v1/tasks` endpoints
4. ✅ System can execute development workflows via `/api/v1/projects/{id}/workflows/{type}`
5. ✅ Multi-agent coordination is demonstrated through task dependencies
6. ✅ Work preservation is shown through checkpointing and recovery

### Technical Requirements
1. ✅ Proper error handling and validation for all API calls
2. ✅ Authentication and authorization implemented correctly
3. ✅ Database schema properly utilized for state management
4. ✅ Task status tracking and progress monitoring
5. ✅ Worker health monitoring and failover handling

### Demonstration Requirements
1. ✅ Complete end-to-end workflow from project creation to task completion
2. ✅ Real-time status updates and progress tracking
3. ✅ Multi-agent coordination and task handoff
4. ✅ Error recovery and checkpoint restoration

## Implementation Guidelines

### API Usage Pattern
```bash
# 1. Authentication
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "challenge_user", "email": "user@example.com", "password": "secure_password"}'

# 2. Project Creation
curl -X POST http://localhost:8080/api/v1/projects \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "Challenge Project", "description": "Multi-agent API challenge", "path": "/challenge", "type": "go"}'

# 3. Task Management
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "Planning Task", "type": "planning", "priority": "high", "parameters": {"requirements": "Create multi-agent system"}}'

# 4. Workflow Execution
curl -X POST http://localhost:8080/api/v1/projects/{projectId}/workflows/planning \
  -H "Authorization: Bearer <token>"
```

### Multi-Agent Coordination Pattern
```go
type Agent interface {
    GetCapabilities() []string
    CanHandle(task Task) bool
    Execute(task Task) (Result, error)
}

type Coordinator struct {
    agents []Agent
    taskManager TaskManager
}

func (c *Coordinator) AssignTask(task Task) error {
    for _, agent := range c.agents {
        if agent.CanHandle(task) {
            return c.taskManager.Assign(task, agent)
        }
    }
    return fmt.Errorf("no suitable agent found for task: %s", task.Name)
}
```

### Task Checkpointing Pattern
```go
type Checkpoint struct {
    TaskID      string                 `json:"task_id"`
    State       map[string]interface{} `json:"state"`
    Progress    float64                `json:"progress"`
    Timestamp   time.Time              `json:"timestamp"`
    Metadata    map[string]interface{} `json:"metadata"`
}

func (t *Task) SaveCheckpoint(state map[string]interface{}, progress float64) error {
    checkpoint := Checkpoint{
        TaskID:    t.ID,
        State:     state,
        Progress:  progress,
        Timestamp: time.Now(),
        Metadata:  t.Metadata,
    }
    return t.taskManager.SaveCheckpoint(checkpoint)
}
```

## Testing Strategy

### Unit Tests
- Test individual agent capabilities and task handling
- Test coordinator task assignment logic
- Test checkpoint save/restore functionality

### Integration Tests
- Test complete API workflow from authentication to task completion
- Test multi-agent coordination scenarios
- Test error recovery and checkpoint restoration

### End-to-End Tests
- Test complete challenge scenario with all components
- Test system behavior under load
- Test recovery from simulated failures

## Evaluation Criteria

### Code Quality (40%)
- Clean, well-documented code
- Proper error handling and validation
- Following Go best practices and HelixCode patterns

### Functionality (30%)
- All success criteria met
- Multi-agent coordination working correctly
- Work preservation and checkpointing implemented

### API Integration (20%)
- Proper use of HelixCode REST API
- Authentication and authorization implemented
- Real-time status and progress tracking

### Documentation (10%)
- Clear setup and usage instructions
- API usage examples
- Challenge demonstration guide

## Getting Started

1. **Setup Environment**
   ```bash
   cd /Volumes/T7/Projects/HelixCode/HelixCode
   export HELIX_DATABASE_PASSWORD=helixcode123
   ./bin/helixcode
   ```

2. **Verify API Access**
   ```bash
   curl http://localhost:8080/health
   ```

3. **Implement Challenge**
   - Start with authentication and project creation
   - Implement multi-agent coordination
   - Add task management and workflow execution
   - Implement checkpointing and work preservation

4. **Test and Validate**
   - Run through complete workflow
   - Verify multi-agent coordination
   - Test checkpointing and recovery

## Bonus Challenges

1. **Advanced Multi-Agent Features**
   - Implement agent learning and capability improvement
   - Add dynamic task re-assignment based on performance
   - Implement agent collaboration on complex tasks

2. **Enhanced Workflow Support**
   - Add custom workflow types
   - Implement workflow templates and reuse
   - Add workflow visualization and monitoring

3. **Performance Optimization**
   - Implement task batching and parallel execution
   - Add worker load balancing
   - Optimize database queries and caching

This challenge demonstrates the core capabilities of HelixCode as a distributed AI development platform while showcasing practical multi-agent system implementation.