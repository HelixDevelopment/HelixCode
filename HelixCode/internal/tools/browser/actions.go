package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

// ActionExecutor executes browser actions
type ActionExecutor interface {
	// Navigate navigates to a URL
	Navigate(ctx context.Context, browserID, url string) error

	// Click clicks an element
	Click(ctx context.Context, browserID string, selector Selector) error

	// Type types text into an element
	Type(ctx context.Context, browserID string, selector Selector, text string, opts *TypeOptions) error

	// Scroll scrolls the page
	Scroll(ctx context.Context, browserID string, opts *ScrollOptions) error

	// Evaluate evaluates JavaScript
	Evaluate(ctx context.Context, browserID, script string) (*EvaluateResult, error)

	// GetElement gets an element
	GetElement(ctx context.Context, browserID string, selector Selector) (*Element, error)

	// GetElements gets multiple elements
	GetElements(ctx context.Context, browserID string, selector Selector) ([]*Element, error)

	// WaitForSelector waits for an element to appear
	WaitForSelector(ctx context.Context, browserID string, selector Selector, timeout time.Duration) error

	// WaitForNavigation waits for navigation to complete
	WaitForNavigation(ctx context.Context, browserID string, timeout time.Duration) error

	// GetPageInfo returns the current page URL and title
	GetPageInfo(ctx context.Context, browserID string) (url, title string, err error)
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
	Delay      time.Duration // Delay between keystrokes
	Clear      bool          // Clear existing content first
	PressEnter bool          // Press Enter after typing
}

// ScrollOptions configures scrolling
type ScrollOptions struct {
	X       int
	Y       int
	Smooth  bool
	Element *Selector // Scroll to element
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

// Rectangle defines a rectangular area
type Rectangle struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// DefaultActionExecutor implements ActionExecutor
type DefaultActionExecutor struct {
	controller Controller
}

// NewDefaultActionExecutor creates a new action executor
func NewDefaultActionExecutor(controller Controller) *DefaultActionExecutor {
	return &DefaultActionExecutor{
		controller: controller,
	}
}

// Navigate navigates to a URL
func (e *DefaultActionExecutor) Navigate(ctx context.Context, browserID, url string) error {
	browserCtx, _, err := e.controller.GetContext(browserID)
	if err != nil {
		return err
	}

	return chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	)
}

// Click clicks an element
func (e *DefaultActionExecutor) Click(ctx context.Context, browserID string, selector Selector) error {
	browserCtx, _, err := e.controller.GetContext(browserID)
	if err != nil {
		return err
	}

	sel := e.buildSelector(selector)
	return chromedp.Run(browserCtx,
		chromedp.WaitVisible(sel),
		chromedp.Click(sel, chromedp.NodeVisible),
	)
}

// Type types text into an element
func (e *DefaultActionExecutor) Type(ctx context.Context, browserID string, selector Selector, text string, opts *TypeOptions) error {
	browserCtx, _, err := e.controller.GetContext(browserID)
	if err != nil {
		return err
	}

	if opts == nil {
		opts = &TypeOptions{}
	}

	sel := e.buildSelector(selector)

	var actions []chromedp.Action

	// Wait for element to be visible
	actions = append(actions, chromedp.WaitVisible(sel))

	// Clear if requested
	if opts.Clear {
		actions = append(actions, chromedp.Clear(sel))
	}

	// Type with or without delay
	if opts.Delay > 0 {
		// Type character by character with delay
		for _, char := range text {
			actions = append(actions,
				chromedp.SendKeys(sel, string(char), chromedp.NodeVisible),
				chromedp.Sleep(opts.Delay),
			)
		}
	} else {
		actions = append(actions,
			chromedp.SendKeys(sel, text, chromedp.NodeVisible),
		)
	}

	// Press Enter if requested
	if opts.PressEnter {
		actions = append(actions,
			chromedp.SendKeys(sel, "\n", chromedp.NodeVisible),
		)
	}

	return chromedp.Run(browserCtx, actions...)
}

// Scroll scrolls the page
func (e *DefaultActionExecutor) Scroll(ctx context.Context, browserID string, opts *ScrollOptions) error {
	browserCtx, _, err := e.controller.GetContext(browserID)
	if err != nil {
		return err
	}

	if opts == nil {
		opts = &ScrollOptions{}
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

// Evaluate evaluates JavaScript
func (e *DefaultActionExecutor) Evaluate(ctx context.Context, browserID, script string) (*EvaluateResult, error) {
	browserCtx, _, err := e.controller.GetContext(browserID)
	if err != nil {
		return nil, err
	}

	var result interface{}
	err = chromedp.Run(browserCtx,
		chromedp.Evaluate(script, &result),
	)

	resultType := "undefined"
	if result != nil {
		resultType = fmt.Sprintf("%T", result)
	}

	return &EvaluateResult{
		Value: result,
		Type:  resultType,
		Error: err,
	}, nil
}

// GetElement gets an element
func (e *DefaultActionExecutor) GetElement(ctx context.Context, browserID string, selector Selector) (*Element, error) {
	browserCtx, _, err := e.controller.GetContext(browserID)
	if err != nil {
		return nil, err
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

	return e.nodeToElement(browserCtx, nodes[0])
}

// GetElements gets multiple elements
func (e *DefaultActionExecutor) GetElements(ctx context.Context, browserID string, selector Selector) ([]*Element, error) {
	browserCtx, _, err := e.controller.GetContext(browserID)
	if err != nil {
		return nil, err
	}

	sel := e.buildSelector(selector)

	var nodes []*cdp.Node
	if err := chromedp.Run(browserCtx,
		chromedp.Nodes(sel, &nodes, chromedp.ByQueryAll),
	); err != nil {
		return nil, err
	}

	elements := make([]*Element, 0, len(nodes))
	for _, node := range nodes {
		elem, err := e.nodeToElement(browserCtx, node)
		if err == nil {
			elements = append(elements, elem)
		}
	}

	return elements, nil
}

// WaitForSelector waits for an element to appear
func (e *DefaultActionExecutor) WaitForSelector(ctx context.Context, browserID string, selector Selector, timeout time.Duration) error {
	browserCtx, _, err := e.controller.GetContext(browserID)
	if err != nil {
		return err
	}

	// Create context with timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		browserCtx, cancel = context.WithTimeout(browserCtx, timeout)
		defer cancel()
	}

	sel := e.buildSelector(selector)
	return chromedp.Run(browserCtx,
		chromedp.WaitVisible(sel),
	)
}

// WaitForNavigation waits for navigation to complete
func (e *DefaultActionExecutor) WaitForNavigation(ctx context.Context, browserID string, timeout time.Duration) error {
	browserCtx, _, err := e.controller.GetContext(browserID)
	if err != nil {
		return err
	}

	// Create context with timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		browserCtx, cancel = context.WithTimeout(browserCtx, timeout)
		defer cancel()
	}

	return chromedp.Run(browserCtx,
		chromedp.WaitReady("body"),
	)
}

// GetPageInfo returns the current page URL and title
func (e *DefaultActionExecutor) GetPageInfo(ctx context.Context, browserID string) (url, title string, err error) {
	browserCtx, _, err := e.controller.GetContext(browserID)
	if err != nil {
		return "", "", err
	}

	err = chromedp.Run(browserCtx,
		chromedp.Location(&url),
		chromedp.Title(&title),
	)

	return url, title, err
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
		// Create XPath for text-based selection
		return fmt.Sprintf("//*[contains(text(),'%s')]", strings.ReplaceAll(selector.Value, "'", "\\'"))
	default:
		return selector.Value
	}
}

// nodeToElement converts a CDP node to an Element
func (e *DefaultActionExecutor) nodeToElement(ctx context.Context, node *cdp.Node) (*Element, error) {
	element := &Element{
		ID:         fmt.Sprintf("%d", node.NodeID),
		TagName:    node.LocalName,
		Attributes: make(map[string]string),
		Text:       node.NodeValue,
	}

	// Parse attributes
	for i := 0; i < len(node.Attributes); i += 2 {
		if i+1 < len(node.Attributes) {
			element.Attributes[node.Attributes[i]] = node.Attributes[i+1]
		}
	}

	// Get text content if node value is empty
	if element.Text == "" && len(node.Children) > 0 {
		var textContent string
		// Try to get text content via JavaScript
		script := fmt.Sprintf(`document.evaluate('//*[@id="%s"]', document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue?.textContent || ''`, element.ID)
		chromedp.Run(ctx,
			chromedp.Evaluate(script, &textContent),
		)
		element.Text = strings.TrimSpace(textContent)
	}

	// Get element bounds
	var bounds *dom.BoxModel
	if err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			bounds, err = dom.GetBoxModel().WithNodeID(node.NodeID).Do(ctx)
			return err
		}),
	); err == nil && bounds != nil && len(bounds.Border) >= 4 {
		// bounds.Border is [x1, y1, x2, y2, x3, y3, x4, y4]
		element.Bounds = Rectangle{
			X:      bounds.Border[0],
			Y:      bounds.Border[1],
			Width:  bounds.Border[4] - bounds.Border[0], // x3 - x1
			Height: bounds.Border[5] - bounds.Border[1], // y3 - y1
		}
	}

	// Check visibility
	if bounds != nil {
		element.Visible = element.Bounds.Width > 0 && element.Bounds.Height > 0
	}

	return element, nil
}

// GetByQuery is a helper function to get a single element by CSS selector
func GetByQuery(ctx context.Context, browserID string, cssSelector string, executor ActionExecutor) (*Element, error) {
	return executor.GetElement(ctx, browserID, Selector{
		Type:  SelectorCSS,
		Value: cssSelector,
	})
}

// GetAllByQuery is a helper function to get all elements by CSS selector
func GetAllByQuery(ctx context.Context, browserID string, cssSelector string, executor ActionExecutor) ([]*Element, error) {
	return executor.GetElements(ctx, browserID, Selector{
		Type:  SelectorCSS,
		Value: cssSelector,
	})
}

// ClickBySelector is a helper function to click an element by CSS selector
func ClickBySelector(ctx context.Context, browserID string, cssSelector string, executor ActionExecutor) error {
	return executor.Click(ctx, browserID, Selector{
		Type:  SelectorCSS,
		Value: cssSelector,
	})
}

// TypeIntoSelector is a helper function to type text into an element by CSS selector
func TypeIntoSelector(ctx context.Context, browserID string, cssSelector string, text string, executor ActionExecutor) error {
	return executor.Type(ctx, browserID, Selector{
		Type:  SelectorCSS,
		Value: cssSelector,
	}, text, &TypeOptions{Clear: true})
}
