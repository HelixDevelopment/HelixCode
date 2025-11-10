// Debugging Workflow Example

package main

import (
	"fmt"

	"dev.helix.code/internal/memory"
	"dev.helix.code/internal/session"
	"dev.helix.code/internal/template"
)

func main() {
	fmt.Println("=== Debugging Workflow ===\n")

	sessionMgr := session.NewManager()
	memoryMgr := memory.NewManager()
	templateMgr := template.NewManager()
	templateMgr.RegisterBuiltinTemplates()

	// Create debugging session
	sess := sessionMgr.Create("debug-memory-leak", session.ModeDebugging, "api-server")
	sess.AddTag("bug")
	sess.AddTag("memory")
	sess.SetMetadata("issue_id", "JIRA-1234")
	sess.SetMetadata("severity", "high")
	sessionMgr.Start(sess.ID)

	conv := memoryMgr.CreateConversation("Debug: Memory Leak")
	conv.SessionID = sess.ID

	// Use Bug Fix template
	buggyCode := `func processRequests() {
	for {
		req := <-requests
		go handleRequest(req)  // Goroutine leak!
	}
}`

	prompt, _ := templateMgr.RenderByName("Bug Fix", map[string]interface{}{
		"language":          "Go",
		"error_message":     "memory usage grows unbounded",
		"code":              buggyCode,
		"expected_behavior": "Stable memory usage",
		"actual_behavior":   "Memory grows continuously",
	})

	memoryMgr.AddMessage(conv.ID, memory.NewUserMessage(prompt))

	// AI suggests fix
	fix := `func processRequests() {
	// Use worker pool to limit goroutines
	workers := make(chan struct{}, 100)

	for {
		req := <-requests
		workers <- struct{}{}
		go func(r Request) {
			defer func() { <-workers }()
			handleRequest(r)
		}(req)
	}
}`

	memoryMgr.AddMessage(conv.ID, memory.NewAssistantMessage(
		fmt.Sprintf("Fixed code:\n```go\n%s\n```", fix),
	))

	sessionMgr.Complete(sess.ID)
	fmt.Printf("Debugging session completed: %s\n", sess.Name)
}
