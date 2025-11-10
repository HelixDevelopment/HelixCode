package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
)

// Controller manages browser instances and sessions
type Controller interface {
	// Launch launches a new browser instance
	Launch(ctx context.Context, opts *LaunchOptions) (*Browser, error)

	// Connect connects to an existing browser instance
	Connect(ctx context.Context, wsURL string) (*Browser, error)

	// GetBrowser returns a browser by ID
	GetBrowser(id string) (*Browser, error)

	// ListBrowsers lists all active browsers
	ListBrowsers() []*Browser

	// Close closes a browser instance
	Close(browserID string) error

	// CloseAll closes all browser instances
	CloseAll() error

	// GetContext returns the chromedp context for a browser
	GetContext(browserID string) (context.Context, context.CancelFunc, error)
}

// Browser represents a browser instance
type Browser struct {
	ID          string
	ProcessID   int
	WSEndpoint  string
	Pages       []*Page
	UserDataDir string
	StartTime   time.Time
	Options     *LaunchOptions
	mu          sync.RWMutex
}

// Page represents a browser page/tab
type Page struct {
	ID        string
	BrowserID string
	URL       string
	Title     string
	Viewport  Viewport
	CreatedAt time.Time
}

// Viewport defines the browser viewport size
type Viewport struct {
	Width  int
	Height int
	Scale  float64
}

// LaunchOptions configures browser launch
type LaunchOptions struct {
	Headless          bool
	Width             int
	Height            int
	UserDataDir       string
	Args              []string
	ExecutablePath    string
	Timeout           time.Duration
	SlowMo            time.Duration // Slow down operations for debugging
	DevTools          bool
	Proxy             string
	IgnoreHTTPSErrors bool
	Incognito         bool
}

// DefaultLaunchOptions returns default launch options
func DefaultLaunchOptions() *LaunchOptions {
	return &LaunchOptions{
		Headless: true,
		Width:    1280,
		Height:   720,
		Timeout:  30 * time.Second,
		Args: []string{
			"--disable-dev-shm-usage",
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-gpu",
		},
	}
}

// browserContext holds the chromedp context for a browser
type browserContext struct {
	ctx         context.Context
	cancel      context.CancelFunc
	allocCtx    context.Context
	allocCancel context.CancelFunc
}

// DefaultController implements Controller
type DefaultController struct {
	browsers  sync.Map // map[string]*Browser
	contexts  sync.Map // map[string]*browserContext
	discovery ChromeDiscovery
	mu        sync.RWMutex
}

// NewDefaultController creates a new default controller
func NewDefaultController(discovery ChromeDiscovery) *DefaultController {
	if discovery == nil {
		discovery = NewDefaultChromeDiscovery()
	}
	return &DefaultController{
		discovery: discovery,
	}
}

// Launch launches a new browser instance
func (c *DefaultController) Launch(ctx context.Context, opts *LaunchOptions) (*Browser, error) {
	if opts == nil {
		opts = DefaultLaunchOptions()
	}

	// Find Chrome if not specified
	if opts.ExecutablePath == "" {
		path, err := c.discovery.FindChrome()
		if err != nil {
			return nil, fmt.Errorf("failed to find Chrome: %w", err)
		}
		opts.ExecutablePath = path
	}

	// Build allocator options
	allocOpts := []chromedp.ExecAllocatorOption{
		chromedp.ExecPath(opts.ExecutablePath),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
	}

	if opts.Headless {
		allocOpts = append(allocOpts, chromedp.Headless)
	}

	if opts.Width > 0 && opts.Height > 0 {
		allocOpts = append(allocOpts,
			chromedp.WindowSize(opts.Width, opts.Height),
		)
	}

	if opts.UserDataDir != "" {
		allocOpts = append(allocOpts, chromedp.UserDataDir(opts.UserDataDir))
	}

	if opts.Proxy != "" {
		allocOpts = append(allocOpts, chromedp.ProxyServer(opts.Proxy))
	}

	if opts.IgnoreHTTPSErrors {
		allocOpts = append(allocOpts, chromedp.Flag("ignore-certificate-errors", true))
	}

	if opts.Incognito {
		allocOpts = append(allocOpts, chromedp.Flag("incognito", true))
	}

	// Add custom args
	for _, arg := range opts.Args {
		// Parse args that might have values
		if len(arg) > 0 && arg[0] == '-' {
			allocOpts = append(allocOpts, chromedp.Flag(arg, true))
		}
	}

	// Create allocator context
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, allocOpts...)

	// Create browser context with timeout
	var browserCtx context.Context
	var browserCancel context.CancelFunc

	if opts.Timeout > 0 {
		browserCtx, browserCancel = context.WithTimeout(allocCtx, opts.Timeout)
	} else {
		browserCtx, browserCancel = context.WithCancel(allocCtx)
	}

	browserCtx, _ = chromedp.NewContext(browserCtx)

	// Launch browser by running an empty action
	if err := chromedp.Run(browserCtx); err != nil {
		browserCancel()
		allocCancel()
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	// Get WebSocket URL (optional - we can work without it)
	var wsURL string
	_ = chromedp.Run(browserCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Get browser websocket URL if available
			// Note: This may not always be available depending on Chrome version
			return nil
		}),
	)

	browser := &Browser{
		ID:          uuid.New().String(),
		WSEndpoint:  wsURL,
		UserDataDir: opts.UserDataDir,
		StartTime:   time.Now(),
		Options:     opts,
		Pages:       make([]*Page, 0),
	}

	// Store browser and context
	c.browsers.Store(browser.ID, browser)
	c.contexts.Store(browser.ID, &browserContext{
		ctx:         browserCtx,
		cancel:      browserCancel,
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
	})

	return browser, nil
}

// Connect connects to an existing browser instance
func (c *DefaultController) Connect(ctx context.Context, wsURL string) (*Browser, error) {
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(ctx, wsURL)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)

	if err := chromedp.Run(browserCtx); err != nil {
		browserCancel()
		allocCancel()
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	browser := &Browser{
		ID:         uuid.New().String(),
		WSEndpoint: wsURL,
		StartTime:  time.Now(),
		Pages:      make([]*Page, 0),
	}

	c.browsers.Store(browser.ID, browser)
	c.contexts.Store(browser.ID, &browserContext{
		ctx:         browserCtx,
		cancel:      browserCancel,
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
	})

	return browser, nil
}

// GetBrowser returns a browser by ID
func (c *DefaultController) GetBrowser(id string) (*Browser, error) {
	val, ok := c.browsers.Load(id)
	if !ok {
		return nil, fmt.Errorf("browser not found: %s", id)
	}
	return val.(*Browser), nil
}

// ListBrowsers lists all active browsers
func (c *DefaultController) ListBrowsers() []*Browser {
	var browsers []*Browser
	c.browsers.Range(func(key, value interface{}) bool {
		browsers = append(browsers, value.(*Browser))
		return true
	})
	return browsers
}

// Close closes a browser instance
func (c *DefaultController) Close(browserID string) error {
	val, ok := c.contexts.LoadAndDelete(browserID)
	if !ok {
		return fmt.Errorf("browser not found: %s", browserID)
	}

	bctx := val.(*browserContext)

	// Cancel contexts to clean up
	if bctx.cancel != nil {
		bctx.cancel()
	}
	if bctx.allocCancel != nil {
		bctx.allocCancel()
	}

	c.browsers.Delete(browserID)

	return nil
}

// CloseAll closes all browser instances
func (c *DefaultController) CloseAll() error {
	var lastErr error

	c.contexts.Range(func(key, value interface{}) bool {
		browserID := key.(string)
		if err := c.Close(browserID); err != nil {
			lastErr = err
		}
		return true
	})

	return lastErr
}

// GetContext returns the chromedp context for a browser
func (c *DefaultController) GetContext(browserID string) (context.Context, context.CancelFunc, error) {
	val, ok := c.contexts.Load(browserID)
	if !ok {
		return nil, nil, fmt.Errorf("browser context not found: %s", browserID)
	}

	bctx := val.(*browserContext)
	return bctx.ctx, bctx.cancel, nil
}

// AddPage adds a page to a browser
func (b *Browser) AddPage(page *Page) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Pages = append(b.Pages, page)
}

// GetPage returns a page by ID
func (b *Browser) GetPage(pageID string) (*Page, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, page := range b.Pages {
		if page.ID == pageID {
			return page, nil
		}
	}

	return nil, fmt.Errorf("page not found: %s", pageID)
}

// RemovePage removes a page from a browser
func (b *Browser) RemovePage(pageID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, page := range b.Pages {
		if page.ID == pageID {
			b.Pages = append(b.Pages[:i], b.Pages[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("page not found: %s", pageID)
}
