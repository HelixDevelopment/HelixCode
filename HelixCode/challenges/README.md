# HelixCode Application Challenges

This directory contains application challenges designed to demonstrate and test HelixCode's distributed AI development capabilities.

## Available Challenges

### Multi-Agent API Challenge

**Description**: Create a distributed multi-agent system using HelixCode's API endpoints that demonstrates intelligent task division, coordination, and execution across different development workflows.

**Files**:
- `multi-agent-api-challenge.md` - Complete challenge specification and requirements
- `multi-agent-api-challenge-solution.go` - Reference implementation demonstrating the solution

**Key Features Demonstrated**:
- Multi-agent coordination and task assignment
- Distributed task execution with dependencies
- Development workflow integration (planning, building, testing)
- Work preservation through checkpointing
- HelixCode REST API integration

## Getting Started

### Prerequisites
- HelixCode server running on `localhost:8080`
- PostgreSQL database with proper credentials
- Go 1.24.0 or later

### Setup Instructions

1. **Start HelixCode Server**
   ```bash
   cd /Volumes/T7/Projects/HelixCode/HelixCode
   export HELIX_DATABASE_PASSWORD=helixcode123
   ./bin/helixcode
   ```

2. **Verify Server Health**
   ```bash
   curl http://localhost:8080/health
   ```
   Should return: `{"status":"healthy","version":"1.0.0","timestamp":"..."}`

3. **Run Challenge Solution**
   ```bash
   cd challenges
   go run multi-agent-api-challenge-solution.go
   ```

## Challenge Structure

### Core Components

1. **Multi-Agent System**
   - Planning Agent: Analyzes requirements and creates task breakdown
   - Building Agent: Handles code generation and integration
   - Testing Agent: Executes tests and validates results
   - Coordination Agent: Manages inter-agent communication

2. **Task Management**
   - Task creation, assignment, and tracking via API
   - Dependency resolution and execution order
   - Status monitoring and progress tracking

3. **Workflow Integration**
   - Planning workflow execution
   - Building workflow execution
   - Testing workflow execution
   - Real-time progress updates

### API Endpoints Used

- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User authentication
- `POST /api/v1/projects` - Project creation
- `POST /api/v1/tasks` - Task creation
- `POST /api/v1/projects/{id}/workflows/{type}` - Workflow execution
- `GET /health` - System health check

## Success Criteria

### Functional Requirements
- ✅ User authentication and session management
- ✅ Project creation and management
- ✅ Multi-agent task coordination
- ✅ Workflow execution and tracking
- ✅ Checkpointing and work preservation

### Technical Requirements
- ✅ Proper error handling and validation
- ✅ API authentication and authorization
- ✅ Database state management
- ✅ Real-time status updates
- ✅ Task dependency resolution

## Testing the Challenge

### Manual Testing
1. Start the HelixCode server
2. Run the challenge solution
3. Monitor console output for progress
4. Verify all success criteria are met

### Automated Testing
```bash
# Run the challenge solution
go run multi-agent-api-challenge-solution.go

# Expected output:
# 🚀 Starting Multi-Agent API Challenge Solution
# 🔐 Authenticating with HelixCode API...
# ✅ Authenticated as user: challenge_user
# 📁 Creating challenge project...
# ✅ Project created with ID: proj_placeholder
# 🤖 Executing multi-agent workflow...
# 📋 Processing task: Requirements Analysis
# 🤖 Assigning task 'Requirements Analysis' to agent *main.PlanningAgent
# 📊 PlanningAgent executing: Requirements Analysis
# ✅ Task Requirements Analysis completed by agent: completed
# ... (similar output for other tasks)
# 🎉 Multi-Agent API Challenge Solution completed successfully!
```

## Customization

### Adding New Agents
```go
type CustomAgent struct {
    ID           string
    Capabilities []string
}

func (a *CustomAgent) GetCapabilities() []string {
    return a.Capabilities
}

func (a *CustomAgent) CanHandle(task Task) bool {
    return task.Type == "custom"
}

func (a *CustomAgent) Execute(task Task) (TaskResult, error) {
    // Custom agent logic here
    return TaskResult{Status: "completed"}, nil
}
```

### Adding New Workflows
```go
func (s *MultiAgentChallengeSolution) executeCustomWorkflow() error {
    // Custom workflow logic
    return nil
}
```

## Troubleshooting

### Common Issues

1. **Server Not Starting**
   - Verify database connection settings
   - Check if port 8080 is available
   - Ensure proper environment variables are set

2. **Authentication Failures**
   - Verify user registration succeeds
   - Check JWT token generation
   - Ensure proper authorization headers

3. **Task Assignment Issues**
   - Verify agent capabilities match task types
   - Check task dependencies are resolved
   - Ensure proper error handling

### Debug Mode
Enable debug logging by setting environment variable:
```bash
export HELIX_DEBUG=true
```

## Next Steps

After completing this challenge, consider:

1. **Advanced Multi-Agent Features**
   - Agent learning and capability improvement
   - Dynamic task re-assignment
   - Agent collaboration on complex tasks

2. **Enhanced Workflow Support**
   - Custom workflow types
   - Workflow templates and reuse
   - Workflow visualization

3. **Performance Optimization**
   - Task batching and parallel execution
   - Worker load balancing
   - Database query optimization

## Contributing

To contribute new challenges:
1. Create challenge specification in markdown format
2. Provide reference implementation
3. Include testing instructions
4. Update this README with challenge details

## License

This challenge material is part of the HelixCode project and follows the same licensing terms.