// Feature Development Example
// Complete workflow for implementing a new feature using Phase 3

package main

import (
	"fmt"
	"log"
	"time"

	"dev.helix.code/internal/memory"
	"dev.helix.code/internal/persistence"
	"dev.helix.code/internal/session"
	"dev.helix.code/internal/template"
)

func main() {
	fmt.Println("=== Feature Development Workflow ===\n")

	// Initialize
	sessionMgr := session.NewManager()
	memoryMgr := memory.NewManager()
	templateMgr := template.NewManager()
	store, err := persistence.NewStore("./data")
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	store.SetSessionManager(sessionMgr)
	store.SetMemoryManager(memoryMgr)
	store.SetTemplateManager(templateMgr)
	store.EnableAutoSave(300)

	templateMgr.RegisterBuiltinTemplates()

	// Phase 1: Planning
	fmt.Println("📋 Phase 1: Planning")
	planningSession, err := sessionMgr.Create(
		"api-server",
		"plan-user-auth",
		"Planning session for user authentication",
		session.ModePlanning,
	)
	if err != nil {
		log.Fatalf("Failed to create planning session: %v", err)
	}
	planningSession.AddTag("authentication")
	planningSession.AddTag("planning")
	sessionMgr.Start(planningSession.ID)

	planConv, err := memoryMgr.CreateConversation("Planning: User Authentication")
	if err != nil {
		log.Fatalf("Failed to create planning conversation: %v", err)
	}
	planConv.SessionID = planningSession.ID

	memoryMgr.AddMessage(planConv.ID, memory.NewUserMessage(
		"I need to design a user authentication system with JWT tokens",
	))

	memoryMgr.AddMessage(planConv.ID, memory.NewAssistantMessage(
		"Let's break this down:\n1. User registration\n2. Login with JWT\n3. Token validation middleware\n4. Token refresh\n5. Logout",
	))

	fmt.Printf("  Created planning session: %s\n", planningSession.Name)
	fmt.Printf("  %d messages in planning conversation\n\n", len(planConv.GetMessages()))

	sessionMgr.Complete(planningSession.ID)
	time.Sleep(100 * time.Millisecond)

	// Phase 2: Implementation
	fmt.Println("🔨 Phase 2: Implementation")
	buildSession, err := sessionMgr.Create(
		"api-server",
		"implement-user-auth",
		"Implementation session for user authentication",
		session.ModeBuilding,
	)
	if err != nil {
		log.Fatalf("Failed to create build session: %v", err)
	}
	buildSession.AddTag("authentication")
	buildSession.AddTag("implementation")
	buildSession.SetMetadata("sprint", "23")
	sessionMgr.Start(buildSession.ID)

	buildConv, err := memoryMgr.CreateConversation("Implementation: User Auth")
	if err != nil {
		log.Fatalf("Failed to create build conversation: %v", err)
	}
	buildConv.SessionID = buildSession.ID

	// Generate login handler from template
	loginHandler, _ := templateMgr.RenderByName("Function", map[string]interface{}{
		"function_name": "HandleLogin",
		"parameters":    "w http.ResponseWriter, r *http.Request",
		"return_type":   "",
		"body": `	var creds Credentials
	json.NewDecoder(r.Body).Decode(&creds)

	token, err := generateJWT(creds.Username)
	if err != nil {
		http.Error(w, "Failed to generate token", 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})`,
	})

	memoryMgr.AddMessage(buildConv.ID, memory.NewAssistantMessage(
		fmt.Sprintf("Login handler:\n```go\n%s\n```", loginHandler),
	))

	fmt.Printf("  Created implementation session: %s\n", buildSession.Name)
	fmt.Printf("  Generated login handler code\n\n")

	sessionMgr.Complete(buildSession.ID)
	time.Sleep(100 * time.Millisecond)

	// Phase 3: Testing
	fmt.Println("🧪 Phase 3: Testing")
	testSession, err := sessionMgr.Create(
		"api-server",
		"test-user-auth",
		"Testing session for user authentication",
		session.ModeTesting,
	)
	if err != nil {
		log.Fatalf("Failed to create test session: %v", err)
	}
	testSession.AddTag("authentication")
	testSession.AddTag("testing")
	sessionMgr.Start(testSession.ID)

	testConv, err := memoryMgr.CreateConversation("Testing: User Auth")
	if err != nil {
		log.Fatalf("Failed to create test conversation: %v", err)
	}
	testConv.SessionID = testSession.ID

	// Generate test from template
	testCode, _ := templateMgr.RenderByName("Function", map[string]interface{}{
		"function_name": "TestHandleLogin",
		"parameters":    "t *testing.T",
		"return_type":   "",
		"body": `	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte("{\"username\":\"test\",\"password\":\"pass\"}")))
	w := httptest.NewRecorder()

	HandleLogin(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "token")`,
	})

	memoryMgr.AddMessage(testConv.ID, memory.NewAssistantMessage(
		fmt.Sprintf("Test code:\n```go\n%s\n```", testCode),
	))

	fmt.Printf("  Created testing session: %s\n", testSession.Name)
	fmt.Printf("  Generated test code\n\n")

	sessionMgr.Complete(testSession.ID)

	// Save all progress
	store.Save()

	// Show summary
	fmt.Println("=== Feature Development Summary ===")
	stats := sessionMgr.GetStatistics()
	fmt.Printf("Total sessions: %d\n", stats.Total)
	fmt.Printf("Completed sessions: %d\n", stats.ByStatus[session.StatusCompleted])
	fmt.Printf("Total conversations: %d\n", len(memoryMgr.GetAll()))
	fmt.Printf("Total messages: %d\n", memoryMgr.TotalMessages())

	// Show all sessions
	fmt.Println("\nSessions created:")
	for _, sess := range sessionMgr.GetByProject("api-server") {
		duration := sess.EndedAt.Sub(sess.StartedAt)
		fmt.Printf("  %s (%s) - %v\n", sess.Name, sess.Mode, duration)
	}
}
