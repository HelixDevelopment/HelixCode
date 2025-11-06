// Package browser provides browser automation and control capabilities for HelixCode.
// This package implements browser automation using Chrome DevTools Protocol (CDP) via
// chromedp, enabling web interaction, screenshot capture, element selection, and
// console monitoring for AI-assisted development workflows.
//
// # Architecture
//
// The package is organized into several key components:
//
//   - Controller: Browser instance and session management
//   - ActionExecutor: Browser actions (click, type, scroll, navigate)
//   - ChromeDiscovery: Chrome/Chromium installation detection
//   - ScreenshotAnnotator: Screenshot capture with element annotation
//   - ConsoleMonitor: Console log and error tracking
//   - ElementSelector: Visual element selection for Claude Computer Use
//
// # Features
//
// Browser Control provides comprehensive automation capabilities:
//
//   - Launch and connect to Chrome/Chromium instances
//   - Multi-tab management
//   - Element interaction (click, type, scroll)
//   - JavaScript execution
//   - Screenshot capture with annotation for Claude
//   - Console monitoring and error tracking
//   - Element selection with visual feedback
//   - Wait for elements and navigation
//
// # Example Usage
//
// Basic browser launch and navigation:
//
//	discovery := browser.NewDefaultChromeDiscovery()
//	controller := browser.NewDefaultController(discovery)
//
//	opts := &browser.LaunchOptions{
//	    Headless: true,
//	    Width:    1280,
//	    Height:   720,
//	}
//
//	b, err := controller.Launch(context.Background(), opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer controller.Close(b.ID)
//
// Execute browser actions:
//
//	executor := browser.NewDefaultActionExecutor(controller)
//
//	// Navigate to a page
//	err = executor.Navigate(ctx, pageID, "https://example.com")
//
//	// Click an element
//	err = executor.Click(ctx, pageID, browser.Selector{
//	    Type:  browser.SelectorCSS,
//	    Value: "#submit-button",
//	})
//
//	// Type text
//	err = executor.Type(ctx, pageID, browser.Selector{
//	    Type:  browser.SelectorID,
//	    Value: "search-input",
//	}, "Hello, World!", &browser.TypeOptions{
//	    Clear: true,
//	    PressEnter: true,
//	})
//
// Take annotated screenshots for Claude Computer Use:
//
//	screenshot, err := executor.Screenshot(ctx, pageID, &browser.ScreenshotOptions{
//	    FullPage: true,
//	    Format:   browser.FormatPNG,
//	})
//
//	annotator := browser.NewScreenshotAnnotator()
//	elements, _ := executor.GetElements(ctx, pageID, browser.Selector{
//	    Type:  browser.SelectorCSS,
//	    Value: "button, a, input",
//	})
//
//	annotated, err := annotator.Annotate(screenshot, elements)
//	os.WriteFile("annotated.png", annotated.Data, 0644)
//
// Monitor console output:
//
//	monitor := browser.NewConsoleMonitor()
//	monitor.Start(browserCtx)
//
//	go func() {
//	    for msg := range monitor.GetMessages() {
//	        fmt.Printf("[%s] %s\n", msg.Type, msg.Text)
//	    }
//	}()
//
//	go func() {
//	    for err := range monitor.GetErrors() {
//	        fmt.Printf("Error at %s:%d - %s\n", err.URL, err.Line, err.Text)
//	    }
//	}()
//
// # Computer Use Integration
//
// This package is designed to integrate with Claude's Computer Use capabilities,
// providing annotated screenshots with element coordinates and bounding boxes that
// Claude can use to understand and interact with web interfaces.
//
// Screenshot annotation includes:
//
//   - Element bounding boxes with coordinates
//   - Interactive element highlighting
//   - Numbered element labels for selection
//   - Full page or viewport capture
//
// # Chrome Discovery
//
// The package automatically discovers Chrome/Chromium installations across platforms:
//
//   - macOS: Google Chrome, Chromium, Brave, Microsoft Edge
//   - Linux: google-chrome, chromium, chromium-browser, snap packages
//   - Windows: Chrome in Program Files and Program Files (x86)
//
// Custom Chrome paths can be specified via LaunchOptions.ExecutablePath.
//
// # Design Inspiration
//
// This implementation is inspired by:
//
//   - Cline's Puppeteer integration: Browser automation and screenshot annotation
//   - Claude Computer Use: Visual element selection and coordinate-based interaction
//   - chromedp: Efficient Chrome DevTools Protocol implementation in Go
//
// # References
//
// Technical Design: /Design/TechnicalDesigns/BrowserControl.md
// chromedp: https://github.com/chromedp/chromedp
// Chrome DevTools Protocol: https://chromedevtools.github.io/devtools-protocol/
package browser
