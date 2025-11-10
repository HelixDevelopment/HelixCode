// Code Review Workflow Example
// Automated code review using sessions, templates, and memory

package main

import (
	"fmt"
	"log"

	"dev.helix.code/internal/memory"
	"dev.helix.code/internal/session"
	"dev.helix.code/internal/template"
)

func main() {
	fmt.Println("=== Code Review Workflow ===\n")

	// Initialize managers
	sessionMgr := session.NewManager()
	memoryMgr := memory.NewManager()
	templateMgr := template.NewManager()

	templateMgr.RegisterBuiltinTemplates()

	// Sample code to review
	sampleCode := `func processPayment(amount float64, cardNumber string) error {
	if amount <= 0 {
		return errors.New("invalid amount")
	}

	// TODO: Add card validation
	// TODO: Add fraud detection
	// TODO: Add transaction logging

	return gateway.Charge(cardNumber, amount)
}`

	// Create code review session
	sess, err := sessionMgr.Create(
		"payment-service",
		"review-payment-processing",
		"Code review session for payment processing",
		session.ModeBuilding, // or add ModeReview
	)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	sess.AddTag("review")
	sess.AddTag("payments")
	sess.SetMetadata("reviewer", "ai-assistant")
	sessionMgr.Start(sess.ID)

	fmt.Printf("Started code review session: %s\n\n", sess.Name)

	// Create conversation
	conv, err := memoryMgr.CreateConversation("Code Review: Payment Processing")
	if err != nil {
		log.Fatalf("Failed to create conversation: %v", err)
	}
	conv.SessionID = sess.ID

	// Generate review prompt from template
	reviewPrompt, err := templateMgr.RenderByName("Code Review", map[string]interface{}{
		"language":    "Go",
		"code":        sampleCode,
		"focus_areas": "security, error handling, best practices",
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Generated review prompt:")
	fmt.Println(reviewPrompt)
	fmt.Println()

	// Add to conversation
	memoryMgr.AddMessage(conv.ID, memory.NewUserMessage(reviewPrompt))

	// Simulate AI review response
	reviewFeedback := `Code Review Findings:

**Security Issues:**
1. Card number is stored in plain text - should be encrypted
2. No input sanitization on card number
3. Missing rate limiting to prevent abuse

**Error Handling:**
1. Gateway errors not properly handled
2. No retry logic for transient failures
3. Missing transaction rollback on failure

**Best Practices:**
1. TODOs should be completed before production
2. Add logging for audit trail
3. Consider using struct for payment details instead of primitives
4. Add unit tests for edge cases

**Recommendations:**
- Implement PCI DSS compliance measures
- Add comprehensive error handling
- Complete TODO items
- Add transaction logging
- Write integration tests`

	memoryMgr.AddMessage(conv.ID, memory.NewAssistantMessage(reviewFeedback))

	fmt.Println("AI Review Feedback:")
	fmt.Println(reviewFeedback)
	fmt.Println()

	// Track review metrics
	sess.SetMetadata("issues_found", "7")
	sess.SetMetadata("severity", "high")
	sess.SetMetadata("status", "needs_revision")

	// Complete review
	sessionMgr.Complete(sess.ID)

	// Show summary
	fmt.Println("=== Review Summary ===")
	fmt.Printf("Session: %s\n", sess.Name)
	fmt.Printf("Duration: %v\n", sess.EndedAt.Sub(sess.StartedAt))
	fmt.Printf("Issues found: %s\n", sess.GetMetadata("issues_found"))
	fmt.Printf("Severity: %s\n", sess.GetMetadata("severity"))
	fmt.Printf("Messages in review: %d\n", len(conv.GetMessages()))

	// Export review for documentation
	fmt.Println("\nExporting review conversation...")
	snapshot, _ := memoryMgr.ExportConversation(conv.ID)
	fmt.Printf("Exported conversation with %d messages\n", len(snapshot.Conversation.GetMessages()))
}
