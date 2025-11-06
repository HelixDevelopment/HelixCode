package browser

import (
	"context"
	"fmt"
	"sync"
)

// Config contains browser control configuration
type Config struct {
	DefaultHeadless       bool
	DefaultWidth          int
	DefaultHeight         int
	MaxConcurrentBrowsers int
	ScreenshotFormat      ImageFormat
	ScreenshotQuality     int
	EnableConsoleMonitor  bool
	ConsoleBufferSize     int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultHeadless:       true,
		DefaultWidth:          1280,
		DefaultHeight:         720,
		MaxConcurrentBrowsers: 5,
		ScreenshotFormat:      FormatPNG,
		ScreenshotQuality:     90,
		EnableConsoleMonitor:  true,
		ConsoleBufferSize:     100,
	}
}

// BrowserTools provides a unified interface for browser automation
type BrowserTools struct {
	config     *Config
	controller Controller
	executor   ActionExecutor
	capture    ScreenshotCapture
	annotator  *ScreenshotAnnotator
	selector   *ElementSelector
	monitors   sync.Map // map[browserID]*ConsoleMonitor
	mu         sync.RWMutex
}

// NewBrowserTools creates a new BrowserTools instance
func NewBrowserTools(config *Config) *BrowserTools {
	if config == nil {
		config = DefaultConfig()
	}

	discovery := NewDefaultChromeDiscovery()
	controller := NewDefaultController(discovery)
	executor := NewDefaultActionExecutor(controller)
	capture := NewDefaultScreenshotCapture(controller, executor)
	annotator := NewScreenshotAnnotator(nil)
	selector := NewElementSelector(executor, capture, annotator)

	return &BrowserTools{
		config:     config,
		controller: controller,
		executor:   executor,
		capture:    capture,
		annotator:  annotator,
		selector:   selector,
	}
}

// LaunchBrowser launches a new browser instance with default or custom options
func (bt *BrowserTools) LaunchBrowser(ctx context.Context, opts *LaunchOptions) (*Browser, error) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	// Check concurrent browser limit
	browsers := bt.controller.ListBrowsers()
	if len(browsers) >= bt.config.MaxConcurrentBrowsers {
		return nil, fmt.Errorf("maximum concurrent browsers (%d) reached", bt.config.MaxConcurrentBrowsers)
	}

	// Apply default options if not provided
	if opts == nil {
		opts = &LaunchOptions{
			Headless: bt.config.DefaultHeadless,
			Width:    bt.config.DefaultWidth,
			Height:   bt.config.DefaultHeight,
		}
	}

	browser, err := bt.controller.Launch(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Start console monitoring if enabled
	if bt.config.EnableConsoleMonitor {
		bt.startConsoleMonitor(browser.ID)
	}

	return browser, nil
}

// ConnectBrowser connects to an existing browser instance
func (bt *BrowserTools) ConnectBrowser(ctx context.Context, wsURL string) (*Browser, error) {
	browser, err := bt.controller.Connect(ctx, wsURL)
	if err != nil {
		return nil, err
	}

	// Start console monitoring if enabled
	if bt.config.EnableConsoleMonitor {
		bt.startConsoleMonitor(browser.ID)
	}

	return browser, nil
}

// CloseBrowser closes a browser instance
func (bt *BrowserTools) CloseBrowser(browserID string) error {
	// Stop console monitoring
	bt.stopConsoleMonitor(browserID)

	return bt.controller.Close(browserID)
}

// CloseAllBrowsers closes all browser instances
func (bt *BrowserTools) CloseAllBrowsers() error {
	browsers := bt.controller.ListBrowsers()

	for _, browser := range browsers {
		bt.stopConsoleMonitor(browser.ID)
	}

	return bt.controller.CloseAll()
}

// Navigate navigates to a URL
func (bt *BrowserTools) Navigate(ctx context.Context, browserID, url string) error {
	return bt.executor.Navigate(ctx, browserID, url)
}

// Click clicks an element
func (bt *BrowserTools) Click(ctx context.Context, browserID string, selector Selector) error {
	return bt.executor.Click(ctx, browserID, selector)
}

// Type types text into an element
func (bt *BrowserTools) Type(ctx context.Context, browserID string, selector Selector, text string, opts *TypeOptions) error {
	return bt.executor.Type(ctx, browserID, selector, text, opts)
}

// Scroll scrolls the page
func (bt *BrowserTools) Scroll(ctx context.Context, browserID string, opts *ScrollOptions) error {
	return bt.executor.Scroll(ctx, browserID, opts)
}

// TakeScreenshot takes a screenshot
func (bt *BrowserTools) TakeScreenshot(ctx context.Context, browserID string, opts *ScreenshotOptions) (*Screenshot, error) {
	if opts == nil {
		opts = &ScreenshotOptions{
			Format:  bt.config.ScreenshotFormat,
			Quality: bt.config.ScreenshotQuality,
		}
	}
	return bt.capture.Capture(ctx, browserID, opts)
}

// TakeAnnotatedScreenshot takes a screenshot with annotated interactive elements
func (bt *BrowserTools) TakeAnnotatedScreenshot(ctx context.Context, browserID string, opts *ScreenshotOptions) (*Screenshot, []*Element, error) {
	return bt.selector.CreateAnnotatedScreenshot(ctx, browserID, opts)
}

// GetInteractiveElements returns all interactive elements on the page
func (bt *BrowserTools) GetInteractiveElements(ctx context.Context, browserID string) ([]*Element, error) {
	return bt.selector.GetInteractiveElements(ctx, browserID)
}

// Evaluate evaluates JavaScript
func (bt *BrowserTools) Evaluate(ctx context.Context, browserID, script string) (*EvaluateResult, error) {
	return bt.executor.Evaluate(ctx, browserID, script)
}

// GetElement gets an element by selector
func (bt *BrowserTools) GetElement(ctx context.Context, browserID string, selector Selector) (*Element, error) {
	return bt.executor.GetElement(ctx, browserID, selector)
}

// GetElements gets multiple elements by selector
func (bt *BrowserTools) GetElements(ctx context.Context, browserID string, selector Selector) ([]*Element, error) {
	return bt.executor.GetElements(ctx, browserID, selector)
}

// WaitForSelector waits for an element to appear
func (bt *BrowserTools) WaitForSelector(ctx context.Context, browserID string, selector Selector, timeout int64) error {
	return bt.executor.WaitForSelector(ctx, browserID, selector, 0)
}

// GetPageInfo returns the current page URL and title
func (bt *BrowserTools) GetPageInfo(ctx context.Context, browserID string) (url, title string, err error) {
	return bt.executor.GetPageInfo(ctx, browserID)
}

// GetConsoleMonitor returns the console monitor for a browser
func (bt *BrowserTools) GetConsoleMonitor(browserID string) (*ConsoleMonitor, error) {
	val, ok := bt.monitors.Load(browserID)
	if !ok {
		return nil, fmt.Errorf("console monitor not found for browser: %s", browserID)
	}
	return val.(*ConsoleMonitor), nil
}

// GetConsoleMessages returns all console messages for a browser
func (bt *BrowserTools) GetConsoleMessages(browserID string) ([]*ConsoleMessage, error) {
	monitor, err := bt.GetConsoleMonitor(browserID)
	if err != nil {
		return nil, err
	}
	return monitor.GetMessageLog(), nil
}

// GetConsoleErrors returns all console errors for a browser
func (bt *BrowserTools) GetConsoleErrors(browserID string) ([]*ConsoleMessage, error) {
	monitor, err := bt.GetConsoleMonitor(browserID)
	if err != nil {
		return nil, err
	}
	return monitor.GetErrorLog(), nil
}

// ClearConsoleLogs clears console logs for a browser
func (bt *BrowserTools) ClearConsoleLogs(browserID string) error {
	monitor, err := bt.GetConsoleMonitor(browserID)
	if err != nil {
		return err
	}
	monitor.ClearLogs()
	return nil
}

// ListBrowsers returns all active browsers
func (bt *BrowserTools) ListBrowsers() []*Browser {
	return bt.controller.ListBrowsers()
}

// GetBrowser returns a browser by ID
func (bt *BrowserTools) GetBrowser(browserID string) (*Browser, error) {
	return bt.controller.GetBrowser(browserID)
}

// startConsoleMonitor starts console monitoring for a browser
func (bt *BrowserTools) startConsoleMonitor(browserID string) {
	browserCtx, _, err := bt.controller.GetContext(browserID)
	if err != nil {
		return
	}

	opts := &ConsoleMonitorOptions{
		MaxLogSize:   1000,
		FilterErrors: true,
		BufferSize:   bt.config.ConsoleBufferSize,
	}

	monitor := NewConsoleMonitor(opts)
	monitor.Start(browserCtx)

	bt.monitors.Store(browserID, monitor)
}

// stopConsoleMonitor stops console monitoring for a browser
func (bt *BrowserTools) stopConsoleMonitor(browserID string) {
	val, ok := bt.monitors.LoadAndDelete(browserID)
	if ok {
		monitor := val.(*ConsoleMonitor)
		monitor.Stop()
	}
}

// BrowserSession represents a browser automation session
type BrowserSession struct {
	Browser     *Browser
	Tools       *BrowserTools
	Context     context.Context
	CancelFunc  context.CancelFunc
	StartURL    string
	CurrentURL  string
	Screenshots []*Screenshot
	Errors      []error
	mu          sync.RWMutex
}

// NewBrowserSession creates a new browser session
func NewBrowserSession(ctx context.Context, tools *BrowserTools, opts *LaunchOptions) (*BrowserSession, error) {
	sessionCtx, cancel := context.WithCancel(ctx)

	browser, err := tools.LaunchBrowser(sessionCtx, opts)
	if err != nil {
		cancel()
		return nil, err
	}

	return &BrowserSession{
		Browser:     browser,
		Tools:       tools,
		Context:     sessionCtx,
		CancelFunc:  cancel,
		Screenshots: make([]*Screenshot, 0),
		Errors:      make([]error, 0),
	}, nil
}

// Navigate navigates to a URL in the session
func (bs *BrowserSession) Navigate(url string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	err := bs.Tools.Navigate(bs.Context, bs.Browser.ID, url)
	if err != nil {
		bs.Errors = append(bs.Errors, err)
		return err
	}

	if bs.StartURL == "" {
		bs.StartURL = url
	}
	bs.CurrentURL = url

	return nil
}

// TakeScreenshot takes a screenshot in the session
func (bs *BrowserSession) TakeScreenshot(opts *ScreenshotOptions) (*Screenshot, error) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	screenshot, err := bs.Tools.TakeScreenshot(bs.Context, bs.Browser.ID, opts)
	if err != nil {
		bs.Errors = append(bs.Errors, err)
		return nil, err
	}

	bs.Screenshots = append(bs.Screenshots, screenshot)
	return screenshot, nil
}

// Close closes the browser session
func (bs *BrowserSession) Close() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	err := bs.Tools.CloseBrowser(bs.Browser.ID)
	bs.CancelFunc()
	return err
}

// GetErrors returns all errors that occurred in the session
func (bs *BrowserSession) GetErrors() []error {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	errors := make([]error, len(bs.Errors))
	copy(errors, bs.Errors)
	return errors
}

// GetScreenshots returns all screenshots taken in the session
func (bs *BrowserSession) GetScreenshots() []*Screenshot {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	screenshots := make([]*Screenshot, len(bs.Screenshots))
	copy(screenshots, bs.Screenshots)
	return screenshots
}
