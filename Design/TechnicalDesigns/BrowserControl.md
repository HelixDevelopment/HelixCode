# BrowserControl - Technical Design Document

## Overview

The BrowserControl module provides browser automation capabilities for HelixCode, enabling interaction with web applications, screenshot capture, element selection, and console monitoring. This design is inspired by Cline's Puppeteer integration, adapted for Go using chromedp or go-rod.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      BrowserControl                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │  Controller  │  │    Action    │  │   Chrome     │         │
│  │              │──│   Executor   │──│  Discovery   │         │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘         │
│         │                  │                                    │
│  ┌──────┴──────────────────┴──────────┐                        │
│  │      SessionManager                │                        │
│  └──────┬──────────────┬──────────────┘                        │
│         │              │                                        │
│  ┌──────┴─────┐  ┌─────┴──────┐                               │
│  │ Screenshot │  │  Console   │                               │
│  │  Handler   │  │  Monitor   │                               │
│  └────────────┘  └────────────┘                               │
│                                                                 │
│  ┌─────────────────────────────────────────────────┐          │
│  │         Element Selector                        │          │
│  │  (CSS, XPath, text-based selection)            │          │
│  └─────────────────────────────────────────────────┘          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           │
                           ▼
                  Chrome/Chromium Browser
                  (via Chrome DevTools Protocol)
```

## Core Interfaces

### Controller Interface

```go
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
    Headless       bool
    Width          int
    Height         int
    UserDataDir    string
    Args           []string
    ExecutablePath string
    Timeout        time.Duration
    SlowMo         time.Duration // Slow down operations for debugging
    DevTools       bool
    Proxy          string
    IgnoreHTTPSErrors bool
}
```

### ActionExecutor Interface

```go
// ActionExecutor executes browser actions
type ActionExecutor interface {
    // Navigate navigates to a URL
    Navigate(ctx context.Context, pageID, url string) error

    // Click clicks an element
    Click(ctx context.Context, pageID string, selector Selector) error

    // Type types text into an element
    Type(ctx context.Context, pageID string, selector Selector, text string, opts *TypeOptions) error

    // Scroll scrolls the page
    Scroll(ctx context.Context, pageID string, opts *ScrollOptions) error

    // Screenshot takes a screenshot
    Screenshot(ctx context.Context, pageID string, opts *ScreenshotOptions) (*Screenshot, error)

    // Evaluate evaluates JavaScript
    Evaluate(ctx context.Context, pageID, script string) (*EvaluateResult, error)

    // GetElement gets an element
    GetElement(ctx context.Context, pageID string, selector Selector) (*Element, error)

    // GetElements gets multiple elements
    GetElements(ctx context.Context, pageID string, selector Selector) ([]*Element, error)

    // WaitForSelector waits for an element to appear
    WaitForSelector(ctx context.Context, pageID string, selector Selector, timeout time.Duration) error

    // WaitForNavigation waits for navigation to complete
    WaitForNavigation(ctx context.Context, pageID string, timeout time.Duration) error
}

// Selector represents an element selector
type Selector struct {
    Type  SelectorType
    Value string
}

// SelectorType defines the type of selector
type SelectorType int

const (
    SelectorCSS SelectorType = iota
    SelectorXPath
    SelectorText
    SelectorID
)

// TypeOptions configures typing behavior
type TypeOptions struct {
    Delay       time.Duration // Delay between keystrokes
    Clear       bool          // Clear existing content first
    PressEnter  bool          // Press Enter after typing
}

// ScrollOptions configures scrolling
type ScrollOptions struct {
    X        int
    Y        int
    Smooth   bool
    Element  *Selector // Scroll to element
}

// ScreenshotOptions configures screenshots
type ScreenshotOptions struct {
    FullPage     bool
    Clip         *Rectangle
    OmitBackground bool
    Quality      int // 0-100 for JPEG
    Format       ImageFormat
}

// Rectangle defines a rectangular area
type Rectangle struct {
    X      float64
    Y      float64
    Width  float64
    Height float64
}

// ImageFormat defines the image format
type ImageFormat int

const (
    FormatPNG ImageFormat = iota
    FormatJPEG
    FormatWebP
)

// Screenshot represents a screenshot
type Screenshot struct {
    Data      []byte
    Format    ImageFormat
    Width     int
    Height    int
    Timestamp time.Time
    PageURL   string
}

// EvaluateResult contains JavaScript evaluation result
type EvaluateResult struct {
    Value interface{}
    Type  string
    Error error
}

// Element represents a DOM element
type Element struct {
    ID         string
    TagName    string
    Attributes map[string]string
    Text       string
    Bounds     Rectangle
    Visible    bool
}
```

### ChromeDiscovery Interface

```go
// ChromeDiscovery discovers Chrome installations
type ChromeDiscovery interface {
    // FindChrome finds Chrome/Chromium executable
    FindChrome() (string, error)

    // FindChromeVersion returns the Chrome version
    FindChromeVersion(path string) (string, error)

    // GetDefaultPaths returns default Chrome paths for the platform
    GetDefaultPaths() []string
}

// ChromeInfo contains Chrome installation info
type ChromeInfo struct {
    Path    string
    Version string
    Type    ChromeType
}

// ChromeType defines the type of Chrome installation
type ChromeType int

const (
    ChromeTypeChrome ChromeType = iota
    ChromeTypeChromium
    ChromeTypeEdge
    ChromeTypeBrave
)
```

## Implementation

### Controller Implementation

```go
// DefaultController implements Controller
type DefaultController struct {
    browsers      sync.Map // map[string]*Browser
    discovery     ChromeDiscovery
    allocatorOpts []chromedp.ExecAllocatorOption
}

// NewDefaultController creates a new default controller
func NewDefaultController(discovery ChromeDiscovery) *DefaultController {
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
        chromedp.DisableGPU,
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

    for _, arg := range opts.Args {
        allocOpts = append(allocOpts, chromedp.Flag(arg, true))
    }

    // Create allocator context
    allocCtx, cancel := chromedp.NewExecAllocator(ctx, allocOpts...)
    defer cancel()

    // Create browser context
    browserCtx, cancel := chromedp.NewContext(allocCtx)
    defer cancel()

    // Launch browser
    if err := chromedp.Run(browserCtx); err != nil {
        return nil, fmt.Errorf("failed to launch browser: %w", err)
    }

    // Get browser info
    var wsURL string
    err := chromedp.Run(browserCtx,
        chromedp.ActionFunc(func(ctx context.Context) error {
            wsURL = chromedp.FromContext(ctx).Target.GetBrowserWebsocketURL()
            return nil
        }),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to get browser info: %w", err)
    }

    browser := &Browser{
        ID:          uuid.New().String(),
        WSEndpoint:  wsURL,
        UserDataDir: opts.UserDataDir,
        StartTime:   time.Now(),
        Options:     opts,
        Pages:       make([]*Page, 0),
    }

    c.browsers.Store(browser.ID, browser)

    return browser, nil
}

// Close closes a browser instance
func (c *DefaultController) Close(browserID string) error {
    val, ok := c.browsers.LoadAndDelete(browserID)
    if !ok {
        return fmt.Errorf("browser not found: %s", browserID)
    }

    browser := val.(*Browser)

    // Close browser context
    // This depends on how we store the context
    // In a real implementation, we'd need to maintain contexts

    return nil
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
```

### Action Executor Implementation

```go
// DefaultActionExecutor implements ActionExecutor
type DefaultActionExecutor struct {
    controller *DefaultController
}

// NewDefaultActionExecutor creates a new action executor
func NewDefaultActionExecutor(controller *DefaultController) *DefaultActionExecutor {
    return &DefaultActionExecutor{
        controller: controller,
    }
}

// Navigate navigates to a URL
func (e *DefaultActionExecutor) Navigate(ctx context.Context, pageID, url string) error {
    // Get browser context for the page
    browserCtx := e.getBrowserContext(pageID)
    if browserCtx == nil {
        return fmt.Errorf("page not found: %s", pageID)
    }

    return chromedp.Run(browserCtx,
        chromedp.Navigate(url),
        chromedp.WaitReady("body"),
    )
}

// Click clicks an element
func (e *DefaultActionExecutor) Click(ctx context.Context, pageID string, selector Selector) error {
    browserCtx := e.getBrowserContext(pageID)
    if browserCtx == nil {
        return fmt.Errorf("page not found: %s", pageID)
    }

    sel := e.buildSelector(selector)
    return chromedp.Run(browserCtx,
        chromedp.Click(sel, chromedp.NodeVisible),
    )
}

// Type types text into an element
func (e *DefaultActionExecutor) Type(ctx context.Context, pageID string, selector Selector, text string, opts *TypeOptions) error {
    browserCtx := e.getBrowserContext(pageID)
    if browserCtx == nil {
        return fmt.Errorf("page not found: %s", pageID)
    }

    if opts == nil {
        opts = &TypeOptions{}
    }

    sel := e.buildSelector(selector)

    var actions []chromedp.Action

    if opts.Clear {
        actions = append(actions,
            chromedp.Clear(sel),
        )
    }

    if opts.Delay > 0 {
        // Type with delay
        for _, char := range text {
            actions = append(actions,
                chromedp.SendKeys(sel, string(char)),
                chromedp.Sleep(opts.Delay),
            )
        }
    } else {
        actions = append(actions,
            chromedp.SendKeys(sel, text),
        )
    }

    if opts.PressEnter {
        actions = append(actions,
            chromedp.SendKeys(sel, "\n"),
        )
    }

    return chromedp.Run(browserCtx, actions...)
}

// Scroll scrolls the page
func (e *DefaultActionExecutor) Scroll(ctx context.Context, pageID string, opts *ScrollOptions) error {
    browserCtx := e.getBrowserContext(pageID)
    if browserCtx == nil {
        return fmt.Errorf("page not found: %s", pageID)
    }

    if opts.Element != nil {
        // Scroll to element
        sel := e.buildSelector(*opts.Element)
        return chromedp.Run(browserCtx,
            chromedp.ScrollIntoView(sel),
        )
    }

    // Scroll by coordinates
    script := fmt.Sprintf("window.scrollTo(%d, %d)", opts.X, opts.Y)
    if opts.Smooth {
        script = fmt.Sprintf("window.scrollTo({left: %d, top: %d, behavior: 'smooth'})", opts.X, opts.Y)
    }

    return chromedp.Run(browserCtx,
        chromedp.Evaluate(script, nil),
    )
}

// Screenshot takes a screenshot
func (e *DefaultActionExecutor) Screenshot(ctx context.Context, pageID string, opts *ScreenshotOptions) (*Screenshot, error) {
    browserCtx := e.getBrowserContext(pageID)
    if browserCtx == nil {
        return nil, fmt.Errorf("page not found: %s", pageID)
    }

    if opts == nil {
        opts = &ScreenshotOptions{
            Format: FormatPNG,
        }
    }

    var buf []byte

    var action chromedp.Action
    if opts.FullPage {
        action = chromedp.FullScreenshot(&buf, int(opts.Quality))
    } else if opts.Clip != nil {
        // Screenshot specific area
        action = chromedp.Screenshot(
            &buf,
            chromedp.ScreenshotClip(opts.Clip.X, opts.Clip.Y, opts.Clip.Width, opts.Clip.Height),
        )
    } else {
        action = chromedp.Screenshot(&buf)
    }

    if err := chromedp.Run(browserCtx, action); err != nil {
        return nil, fmt.Errorf("failed to take screenshot: %w", err)
    }

    // Get current page URL
    var url string
    chromedp.Run(browserCtx,
        chromedp.Location(&url),
    )

    return &Screenshot{
        Data:      buf,
        Format:    opts.Format,
        Timestamp: time.Now(),
        PageURL:   url,
    }, nil
}

// Evaluate evaluates JavaScript
func (e *DefaultActionExecutor) Evaluate(ctx context.Context, pageID, script string) (*EvaluateResult, error) {
    browserCtx := e.getBrowserContext(pageID)
    if browserCtx == nil {
        return nil, fmt.Errorf("page not found: %s", pageID)
    }

    var result interface{}
    err := chromedp.Run(browserCtx,
        chromedp.Evaluate(script, &result),
    )

    return &EvaluateResult{
        Value: result,
        Error: err,
    }, nil
}

// GetElement gets an element
func (e *DefaultActionExecutor) GetElement(ctx context.Context, pageID string, selector Selector) (*Element, error) {
    browserCtx := e.getBrowserContext(pageID)
    if browserCtx == nil {
        return nil, fmt.Errorf("page not found: %s", pageID)
    }

    sel := e.buildSelector(selector)

    var nodes []*cdp.Node
    if err := chromedp.Run(browserCtx,
        chromedp.Nodes(sel, &nodes, chromedp.ByQuery),
    ); err != nil {
        return nil, err
    }

    if len(nodes) == 0 {
        return nil, fmt.Errorf("element not found")
    }

    node := nodes[0]

    element := &Element{
        ID:         fmt.Sprintf("%d", node.NodeID),
        TagName:    node.LocalName,
        Attributes: make(map[string]string),
        Text:       node.NodeValue,
    }

    // Parse attributes
    for i := 0; i < len(node.Attributes); i += 2 {
        element.Attributes[node.Attributes[i]] = node.Attributes[i+1]
    }

    return element, nil
}

// buildSelector builds a chromedp selector from Selector
func (e *DefaultActionExecutor) buildSelector(selector Selector) string {
    switch selector.Type {
    case SelectorCSS:
        return selector.Value
    case SelectorXPath:
        return selector.Value
    case SelectorID:
        return "#" + selector.Value
    case SelectorText:
        return fmt.Sprintf("//*[contains(text(),'%s')]", selector.Value)
    default:
        return selector.Value
    }
}

// getBrowserContext gets the browser context for a page
func (e *DefaultActionExecutor) getBrowserContext(pageID string) context.Context {
    // This would retrieve the context from storage
    // Simplified for this example
    return nil
}
```

### Chrome Discovery Implementation

```go
// DefaultChromeDiscovery implements ChromeDiscovery
type DefaultChromeDiscovery struct{}

// NewDefaultChromeDiscovery creates a new Chrome discovery
func NewDefaultChromeDiscovery() *DefaultChromeDiscovery {
    return &DefaultChromeDiscovery{}
}

// FindChrome finds Chrome/Chromium executable
func (d *DefaultChromeDiscovery) FindChrome() (string, error) {
    paths := d.GetDefaultPaths()

    for _, path := range paths {
        if _, err := os.Stat(path); err == nil {
            return path, nil
        }
    }

    // Try PATH
    path, err := exec.LookPath("google-chrome")
    if err == nil {
        return path, nil
    }

    path, err = exec.LookPath("chromium")
    if err == nil {
        return path, nil
    }

    return "", fmt.Errorf("Chrome not found")
}

// GetDefaultPaths returns default Chrome paths for the platform
func (d *DefaultChromeDiscovery) GetDefaultPaths() []string {
    switch runtime.GOOS {
    case "darwin": // macOS
        return []string{
            "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
            "/Applications/Chromium.app/Contents/MacOS/Chromium",
            "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
            "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
        }
    case "linux":
        return []string{
            "/usr/bin/google-chrome",
            "/usr/bin/chromium",
            "/usr/bin/chromium-browser",
            "/snap/bin/chromium",
        }
    case "windows":
        return []string{
            `C:\Program Files\Google\Chrome\Application\chrome.exe`,
            `C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
            `C:\Program Files\Chromium\Application\chrome.exe`,
        }
    default:
        return []string{}
    }
}

// FindChromeVersion returns the Chrome version
func (d *DefaultChromeDiscovery) FindChromeVersion(path string) (string, error) {
    cmd := exec.Command(path, "--version")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }

    // Parse version from output
    // Example: "Google Chrome 120.0.6099.109"
    version := strings.TrimSpace(string(output))
    parts := strings.Fields(version)
    if len(parts) < 2 {
        return "", fmt.Errorf("unexpected version format")
    }

    return parts[len(parts)-1], nil
}
```

## Screenshot Features

### Screenshot Annotation

```go
// ScreenshotAnnotator annotates screenshots
type ScreenshotAnnotator struct{}

// Annotate annotates a screenshot with elements
func (sa *ScreenshotAnnotator) Annotate(screenshot *Screenshot, elements []*Element) (*Screenshot, error) {
    // Decode image
    img, _, err := image.Decode(bytes.NewReader(screenshot.Data))
    if err != nil {
        return nil, fmt.Errorf("failed to decode image: %w", err)
    }

    // Create new RGBA image for drawing
    rgba := image.NewRGBA(img.Bounds())
    draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)

    // Draw rectangles around elements
    for _, elem := range elements {
        sa.drawRectangle(rgba, elem.Bounds, color.RGBA{R: 255, G: 0, B: 0, A: 255})
    }

    // Encode back to bytes
    var buf bytes.Buffer
    if err := png.Encode(&buf, rgba); err != nil {
        return nil, fmt.Errorf("failed to encode image: %w", err)
    }

    return &Screenshot{
        Data:      buf.Bytes(),
        Format:    FormatPNG,
        Width:     screenshot.Width,
        Height:    screenshot.Height,
        Timestamp: time.Now(),
        PageURL:   screenshot.PageURL,
    }, nil
}

// drawRectangle draws a rectangle on an image
func (sa *ScreenshotAnnotator) drawRectangle(img *image.RGBA, bounds Rectangle, color color.Color) {
    x1 := int(bounds.X)
    y1 := int(bounds.Y)
    x2 := int(bounds.X + bounds.Width)
    y2 := int(bounds.Y + bounds.Height)

    // Draw top and bottom lines
    for x := x1; x <= x2; x++ {
        img.Set(x, y1, color)
        img.Set(x, y2, color)
    }

    // Draw left and right lines
    for y := y1; y <= y2; y++ {
        img.Set(x1, y, color)
        img.Set(x2, y, color)
    }
}
```

### Element Selector with Screenshot

```go
// ElementSelector helps select elements visually
type ElementSelector struct {
    executor  ActionExecutor
    annotator *ScreenshotAnnotator
}

// SelectElement helps user select an element visually
func (es *ElementSelector) SelectElement(ctx context.Context, pageID string) (*Element, error) {
    // Get all interactive elements
    elements, err := es.executor.GetElements(ctx, pageID, Selector{
        Type:  SelectorCSS,
        Value: "a, button, input, select, textarea, [role='button']",
    })
    if err != nil {
        return nil, err
    }

    // Take screenshot
    screenshot, err := es.executor.Screenshot(ctx, pageID, &ScreenshotOptions{
        FullPage: false,
    })
    if err != nil {
        return nil, err
    }

    // Annotate with element numbers
    annotated, err := es.annotator.Annotate(screenshot, elements)
    if err != nil {
        return nil, err
    }

    // Save annotated screenshot
    filename := fmt.Sprintf("element_selector_%d.png", time.Now().Unix())
    if err := os.WriteFile(filename, annotated.Data, 0644); err != nil {
        return nil, err
    }

    // Present to user for selection
    fmt.Printf("Screenshot saved to: %s\n", filename)
    fmt.Printf("Found %d interactive elements\n", len(elements))
    fmt.Print("Select element number: ")

    var choice int
    fmt.Scanln(&choice)

    if choice < 0 || choice >= len(elements) {
        return nil, fmt.Errorf("invalid choice: %d", choice)
    }

    return elements[choice], nil
}
```

## Console Monitoring

### Console Monitor

```go
// ConsoleMonitor monitors browser console
type ConsoleMonitor struct {
    messages chan *ConsoleMessage
    errors   chan *ConsoleMessage
}

// ConsoleMessage represents a console message
type ConsoleMessage struct {
    Type      ConsoleMessageType
    Text      string
    URL       string
    Line      int
    Column    int
    Timestamp time.Time
}

// ConsoleMessageType defines console message types
type ConsoleMessageType int

const (
    ConsoleLog ConsoleMessageType = iota
    ConsoleInfo
    ConsoleWarning
    ConsoleError
    ConsoleDebug
)

// NewConsoleMonitor creates a new console monitor
func NewConsoleMonitor() *ConsoleMonitor {
    return &ConsoleMonitor{
        messages: make(chan *ConsoleMessage, 100),
        errors:   make(chan *ConsoleMessage, 100),
    }
}

// Start starts monitoring console
func (cm *ConsoleMonitor) Start(ctx context.Context) {
    chromedp.ListenTarget(ctx, func(ev interface{}) {
        switch ev := ev.(type) {
        case *runtime.EventConsoleAPICalled:
            msg := &ConsoleMessage{
                Type:      cm.mapConsoleType(ev.Type),
                Timestamp: time.Now(),
            }

            if len(ev.Args) > 0 {
                msg.Text = ev.Args[0].Description
            }

            cm.messages <- msg

            if msg.Type == ConsoleError {
                cm.errors <- msg
            }

        case *runtime.EventExceptionThrown:
            msg := &ConsoleMessage{
                Type:      ConsoleError,
                Text:      ev.ExceptionDetails.Text,
                Timestamp: time.Now(),
            }

            if ev.ExceptionDetails.URL != "" {
                msg.URL = ev.ExceptionDetails.URL
                msg.Line = int(ev.ExceptionDetails.LineNumber)
                msg.Column = int(ev.ExceptionDetails.ColumnNumber)
            }

            cm.errors <- msg
        }
    })
}

// GetMessages returns the messages channel
func (cm *ConsoleMonitor) GetMessages() <-chan *ConsoleMessage {
    return cm.messages
}

// GetErrors returns the errors channel
func (cm *ConsoleMonitor) GetErrors() <-chan *ConsoleMessage {
    return cm.errors
}

// mapConsoleType maps CDP console type to our type
func (cm *ConsoleMonitor) mapConsoleType(cdpType runtime.APIType) ConsoleMessageType {
    switch cdpType {
    case runtime.APITypeLog:
        return ConsoleLog
    case runtime.APITypeInfo:
        return ConsoleInfo
    case runtime.APITypeWarning:
        return ConsoleWarning
    case runtime.APITypeError:
        return ConsoleError
    case runtime.APITypeDebug:
        return ConsoleDebug
    default:
        return ConsoleLog
    }
}
```

## Testing Strategy

### Unit Tests

```go
// TestChromeDiscovery tests Chrome discovery
func TestChromeDiscovery(t *testing.T) {
    discovery := NewDefaultChromeDiscovery()

    t.Run("find chrome", func(t *testing.T) {
        path, err := discovery.FindChrome()
        if err != nil {
            t.Skip("Chrome not installed")
        }
        assert.NotEmpty(t, path)
        assert.FileExists(t, path)
    })

    t.Run("get default paths", func(t *testing.T) {
        paths := discovery.GetDefaultPaths()
        assert.NotEmpty(t, paths)
    })
}

// TestBrowserLaunch tests browser launch
func TestBrowserLaunch(t *testing.T) {
    discovery := NewDefaultChromeDiscovery()
    controller := NewDefaultController(discovery)

    opts := &LaunchOptions{
        Headless: true,
        Width:    1280,
        Height:   720,
    }

    browser, err := controller.Launch(context.Background(), opts)
    require.NoError(t, err)
    defer controller.Close(browser.ID)

    assert.NotEmpty(t, browser.ID)
    assert.NotEmpty(t, browser.WSEndpoint)
}

// TestNavigation tests page navigation
func TestNavigation(t *testing.T) {
    discovery := NewDefaultChromeDiscovery()
    controller := NewDefaultController(discovery)
    executor := NewDefaultActionExecutor(controller)

    browser, err := controller.Launch(context.Background(), &LaunchOptions{
        Headless: true,
    })
    require.NoError(t, err)
    defer controller.Close(browser.ID)

    // Create page
    pageID := "test-page"

    // Navigate
    err = executor.Navigate(context.Background(), pageID, "https://example.com")
    assert.NoError(t, err)
}

// TestScreenshot tests screenshot capture
func TestScreenshot(t *testing.T) {
    discovery := NewDefaultChromeDiscovery()
    controller := NewDefaultController(discovery)
    executor := NewDefaultActionExecutor(controller)

    browser, err := controller.Launch(context.Background(), &LaunchOptions{
        Headless: true,
    })
    require.NoError(t, err)
    defer controller.Close(browser.ID)

    pageID := "test-page"

    err = executor.Navigate(context.Background(), pageID, "https://example.com")
    require.NoError(t, err)

    screenshot, err := executor.Screenshot(context.Background(), pageID, &ScreenshotOptions{
        Format: FormatPNG,
    })
    require.NoError(t, err)
    assert.NotEmpty(t, screenshot.Data)
    assert.Equal(t, FormatPNG, screenshot.Format)
}
```

### Integration Tests

```go
// TestBrowserControlIntegration tests full browser control workflow
func TestBrowserControlIntegration(t *testing.T) {
    discovery := NewDefaultChromeDiscovery()
    controller := NewDefaultController(discovery)
    executor := NewDefaultActionExecutor(controller)

    browser, err := controller.Launch(context.Background(), &LaunchOptions{
        Headless: true,
        Width:    1280,
        Height:   720,
    })
    require.NoError(t, err)
    defer controller.Close(browser.ID)

    pageID := "test-page"

    t.Run("navigate and interact", func(t *testing.T) {
        // Navigate to a test page
        err := executor.Navigate(context.Background(), pageID, "https://example.com")
        require.NoError(t, err)

        // Take screenshot
        screenshot, err := executor.Screenshot(context.Background(), pageID, nil)
        require.NoError(t, err)
        assert.NotEmpty(t, screenshot.Data)

        // Evaluate JavaScript
        result, err := executor.Evaluate(context.Background(), pageID, "document.title")
        require.NoError(t, err)
        assert.NotNil(t, result.Value)
    })
}
```

## Configuration

```go
// Config contains browser control configuration
type Config struct {
    DefaultHeadless    bool
    DefaultWidth       int
    DefaultHeight      int
    DefaultTimeout     time.Duration
    UserDataDir        string
    KeepUserDataDir    bool
    MaxConcurrentBrowsers int
    ScreenshotFormat   ImageFormat
    ScreenshotQuality  int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
    return &Config{
        DefaultHeadless:       true,
        DefaultWidth:          1280,
        DefaultHeight:         720,
        DefaultTimeout:        30 * time.Second,
        MaxConcurrentBrowsers: 5,
        ScreenshotFormat:      FormatPNG,
        ScreenshotQuality:     90,
    }
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
        },
    }
}
```

## References

### Cline's Browser Integration

- **Location**: `src/core/browser/BrowserController.ts`
- **Features**:
  - Puppeteer-based automation
  - Screenshot capture with annotation
  - Console monitoring
  - Element selection

### Go Libraries

- **chromedp**: `github.com/chromedp/chromedp` - Chrome DevTools Protocol
- **go-rod**: `github.com/go-rod/rod` - Alternative browser automation
- CDP specification: https://chromedevtools.github.io/devtools-protocol/

## Future Enhancements

1. **Video Recording**: Record browser sessions
2. **Network Interception**: Intercept and modify network requests
3. **Cookie Management**: Advanced cookie handling
4. **Local Storage Access**: Read/write local storage
5. **File Upload/Download**: Handle file operations
6. **PDF Generation**: Generate PDFs from pages
7. **Mobile Emulation**: Emulate mobile devices
8. **Geolocation**: Mock geolocation
9. **Permissions**: Manage browser permissions
10. **Multiple Tabs**: Better multi-tab support
11. **Browser Profiles**: User profiles management
12. **Extension Support**: Load browser extensions
