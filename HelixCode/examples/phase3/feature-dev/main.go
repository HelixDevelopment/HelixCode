// Feature Development Example
// Complete workflow for implementing a new feature using Phase 3

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/helixcode/helixcode/internal/memory"
	"github.com/helixcode/helixcode/internal/persistence"
	"github.com/helixcode/helixcode/internal/session"
	"github.com/helixcode/helixcode/internal/template"
)

func main() {
	fmt.Println("=== Feature Development Workflow ===\n")

	// Initialize
	sessionMgr := session.NewManager()
	memoryMgr := memory.NewManager()
	templateMgr := template.NewManager()
	store := persistence.NewStore("./data")

	store.SetSessionManager(sessionMgr)
	store.SetMemoryManager(memoryMgr)
	store.SetTemplateManager(templateMgr)
	store.EnableAutoSave(300)

	templateMgr.RegisterBuiltinTemplates()

	// Phase 1: Planning
	fmt.Println("📋 Phase 1: Planning")
	planningSession := sessionMgr.Create(
		"plan-user-auth",
		session.ModePlanning,
		"api-server",
	)
	planningSession.AddTag("authentication")
	planningSession.AddTag("planning")
	sessionMgr.Start(planningSession.ID)

	planConv := memoryMgr.CreateConversation("Planning: User Authentication")
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
	buildSession := sessionMgr.Create(
		"implement-user-auth",
		session.ModeBuilding,
		"api-server",
	)
	buildSession.AddTag("authentication")
	buildSession.AddTag("implementation")
	buildSession.SetMetadata("sprint", "23")
	sessionMgr.Start(buildSession.ID)

	buildConv := memoryMgr.CreateConversation("Implementation: User Auth")
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
	testSession := sessionMgr.Create(
		"test-user-auth",
		session.ModeTesting,
		"api-server",
	)
	testSession.AddTag("authentication")
	testSession.AddTag("testing")
	sessionMgr.Start(testSession.ID)

	testConv := memoryMgr.CreateConversation("Testing: User Auth")
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
	fmt.Printf("Total sessions: %d\n", stats.TotalSessions)
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
