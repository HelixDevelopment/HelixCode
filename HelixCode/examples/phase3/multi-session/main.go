// Multi-Session Workflow Example

package main

import (
	"fmt"
	"log"
	"time"

	"dev.helix.code/internal/session"
)

func main() {
	fmt.Println("=== Multi-Session Workflow ===")

	mgr := session.NewManager()

	// Working on multiple features
	authSess, err := mgr.Create("project1", "implement-auth", "api", session.ModeBuilding)
	if err != nil {
		log.Fatal(err)
	}
	authSess.AddTag("auth")

	paymentSess, err := mgr.Create("project1", "implement-payments", "api", session.ModeBuilding)
	if err != nil {
		log.Fatal(err)
	}
	paymentSess.AddTag("payments")

	// Start auth work
	fmt.Println("Starting auth work...")
	mgr.Start(authSess.ID)
	time.Sleep(100 * time.Millisecond)

	// Switch to payments (urgent)
	fmt.Println("Pausing auth, switching to payments...")
	mgr.Pause(authSess.ID)
	mgr.Start(paymentSess.ID)
	time.Sleep(100 * time.Millisecond)

	// Complete payments
	fmt.Println("Completed payments work")
	mgr.Complete(paymentSess.ID)

	// Resume auth
	fmt.Println("Resuming auth work...")
	mgr.Resume(authSess.ID)
	time.Sleep(100 * time.Millisecond)

	// Complete auth
	mgr.Complete(authSess.ID)

	// Show all sessions
	fmt.Println("\n=== Sessions Summary ===")
	for _, s := range mgr.GetAll() {
		fmt.Printf("%s (%s): %s\n", s.Name, s.Mode, s.Status)
	}
}
