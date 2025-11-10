package browser

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"time"

	"github.com/chromedp/chromedp"
)

// ImageFormat defines the image format
type ImageFormat int

const (
	FormatPNG ImageFormat = iota
	FormatJPEG
	FormatWebP
)

// String returns the string representation of ImageFormat
func (f ImageFormat) String() string {
	switch f {
	case FormatPNG:
		return "png"
	case FormatJPEG:
		return "jpeg"
	case FormatWebP:
		return "webp"
	default:
		return "unknown"
	}
}

// ScreenshotOptions configures screenshots
type ScreenshotOptions struct {
	FullPage       bool
	Clip           *Rectangle
	OmitBackground bool
	Quality        int // 0-100 for JPEG
	Format         ImageFormat
}

// Screenshot represents a screenshot
type Screenshot struct {
	Data      []byte
	Format    ImageFormat
	Width     int
	Height    int
	Timestamp time.Time
	PageURL   string
}

// ScreenshotCapture handles screenshot capture
type ScreenshotCapture interface {
	// Capture takes a screenshot
	Capture(ctx context.Context, browserID string, opts *ScreenshotOptions) (*Screenshot, error)
}

// DefaultScreenshotCapture implements ScreenshotCapture
type DefaultScreenshotCapture struct {
	controller Controller
	executor   ActionExecutor
}

// NewDefaultScreenshotCapture creates a new screenshot capture
func NewDefaultScreenshotCapture(controller Controller, executor ActionExecutor) *DefaultScreenshotCapture {
	return &DefaultScreenshotCapture{
		controller: controller,
		executor:   executor,
	}
}

// Capture takes a screenshot
func (s *DefaultScreenshotCapture) Capture(ctx context.Context, browserID string, opts *ScreenshotOptions) (*Screenshot, error) {
	browserCtx, _, err := s.controller.GetContext(browserID)
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &ScreenshotOptions{
			Format: FormatPNG,
		}
	}

	var buf []byte
	var actions []chromedp.Action

	if opts.FullPage {
		actions = append(actions, chromedp.FullScreenshot(&buf, int(opts.Quality)))
	} else if opts.Clip != nil {
		// Screenshot specific area with clipping
		actions = append(actions, chromedp.CaptureScreenshot(&buf))
	} else {
		actions = append(actions, chromedp.CaptureScreenshot(&buf))
	}

	if err := chromedp.Run(browserCtx, actions...); err != nil {
		return nil, fmt.Errorf("failed to take screenshot: %w", err)
	}

	// Get current page URL
	url, _, err := s.executor.GetPageInfo(ctx, browserID)
	if err != nil {
		url = ""
	}

	// Decode image to get dimensions
	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("failed to decode screenshot: %w", err)
	}

	bounds := img.Bounds()

	return &Screenshot{
		Data:      buf,
		Format:    opts.Format,
		Width:     bounds.Dx(),
		Height:    bounds.Dy(),
		Timestamp: time.Now(),
		PageURL:   url,
	}, nil
}

// ScreenshotAnnotator annotates screenshots with element information
type ScreenshotAnnotator struct {
	borderColor  color.Color
	labelColor   color.Color
	labelBgColor color.Color
	borderWidth  int
	showLabels   bool
	showBounds   bool
}

// AnnotationOptions configures screenshot annotation
type AnnotationOptions struct {
	BorderColor  color.Color
	LabelColor   color.Color
	LabelBgColor color.Color
	BorderWidth  int
	ShowLabels   bool
	ShowBounds   bool
}

// DefaultAnnotationOptions returns default annotation options
func DefaultAnnotationOptions() *AnnotationOptions {
	return &AnnotationOptions{
		BorderColor:  color.RGBA{R: 255, G: 0, B: 0, A: 255},     // Red
		LabelColor:   color.RGBA{R: 255, G: 255, B: 255, A: 255}, // White
		LabelBgColor: color.RGBA{R: 255, G: 0, B: 0, A: 200},     // Semi-transparent red
		BorderWidth:  2,
		ShowLabels:   true,
		ShowBounds:   true,
	}
}

// NewScreenshotAnnotator creates a new screenshot annotator
func NewScreenshotAnnotator(opts *AnnotationOptions) *ScreenshotAnnotator {
	if opts == nil {
		opts = DefaultAnnotationOptions()
	}

	return &ScreenshotAnnotator{
		borderColor:  opts.BorderColor,
		labelColor:   opts.LabelColor,
		labelBgColor: opts.LabelBgColor,
		borderWidth:  opts.BorderWidth,
		showLabels:   opts.ShowLabels,
		showBounds:   opts.ShowBounds,
	}
}

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

	// Draw rectangles and labels for each element
	for i, elem := range elements {
		if !elem.Visible {
			continue
		}

		if sa.showBounds {
			sa.drawRectangle(rgba, elem.Bounds)
		}

		if sa.showLabels {
			label := fmt.Sprintf("%d", i+1)
			sa.drawLabel(rgba, elem.Bounds, label)
		}
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

// AnnotateWithCoordinates annotates a screenshot with coordinate grid
func (sa *ScreenshotAnnotator) AnnotateWithCoordinates(screenshot *Screenshot, gridSize int) (*Screenshot, error) {
	// Decode image
	img, _, err := image.Decode(bytes.NewReader(screenshot.Data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Create new RGBA image for drawing
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)

	bounds := img.Bounds()
	gridColor := color.RGBA{R: 128, G: 128, B: 128, A: 100} // Semi-transparent gray

	// Draw vertical grid lines
	for x := 0; x < bounds.Dx(); x += gridSize {
		sa.drawVerticalLine(rgba, x, gridColor)
	}

	// Draw horizontal grid lines
	for y := 0; y < bounds.Dy(); y += gridSize {
		sa.drawHorizontalLine(rgba, y, gridColor)
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
func (sa *ScreenshotAnnotator) drawRectangle(img *image.RGBA, bounds Rectangle) {
	x1 := int(bounds.X)
	y1 := int(bounds.Y)
	x2 := int(bounds.X + bounds.Width)
	y2 := int(bounds.Y + bounds.Height)

	// Ensure coordinates are within image bounds
	imgBounds := img.Bounds()
	if x1 < imgBounds.Min.X {
		x1 = imgBounds.Min.X
	}
	if y1 < imgBounds.Min.Y {
		y1 = imgBounds.Min.Y
	}
	if x2 > imgBounds.Max.X {
		x2 = imgBounds.Max.X
	}
	if y2 > imgBounds.Max.Y {
		y2 = imgBounds.Max.Y
	}

	// Draw rectangle with specified border width
	for w := 0; w < sa.borderWidth; w++ {
		// Draw top and bottom lines
		for x := x1; x <= x2; x++ {
			if y1+w < imgBounds.Max.Y {
				img.Set(x, y1+w, sa.borderColor)
			}
			if y2-w >= imgBounds.Min.Y {
				img.Set(x, y2-w, sa.borderColor)
			}
		}

		// Draw left and right lines
		for y := y1; y <= y2; y++ {
			if x1+w < imgBounds.Max.X {
				img.Set(x1+w, y, sa.borderColor)
			}
			if x2-w >= imgBounds.Min.X {
				img.Set(x2-w, y, sa.borderColor)
			}
		}
	}
}

// drawLabel draws a label on an image
func (sa *ScreenshotAnnotator) drawLabel(img *image.RGBA, bounds Rectangle, label string) {
	x := int(bounds.X) + 5
	y := int(bounds.Y) + 5

	// Ensure coordinates are within image bounds
	imgBounds := img.Bounds()
	if x < imgBounds.Min.X || x > imgBounds.Max.X-20 {
		return
	}
	if y < imgBounds.Min.Y || y > imgBounds.Max.Y-15 {
		return
	}

	// Draw simple background rectangle for label
	labelWidth := len(label) * 7
	labelHeight := 12

	for ly := y; ly < y+labelHeight && ly < imgBounds.Max.Y; ly++ {
		for lx := x; lx < x+labelWidth && lx < imgBounds.Max.X; lx++ {
			img.Set(lx, ly, sa.labelBgColor)
		}
	}

	// Note: Drawing actual text would require a font library
	// For now, we just draw the background. In a real implementation,
	// you would use golang.org/x/image/font to draw the text.
}

// drawVerticalLine draws a vertical line
func (sa *ScreenshotAnnotator) drawVerticalLine(img *image.RGBA, x int, color color.Color) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		img.Set(x, y, color)
	}
}

// drawHorizontalLine draws a horizontal line
func (sa *ScreenshotAnnotator) drawHorizontalLine(img *image.RGBA, y int, color color.Color) {
	bounds := img.Bounds()
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		img.Set(x, y, color)
	}
}

// ElementSelector helps select elements visually for Claude Computer Use
type ElementSelector struct {
	executor  ActionExecutor
	capture   ScreenshotCapture
	annotator *ScreenshotAnnotator
}

// NewElementSelector creates a new element selector
func NewElementSelector(executor ActionExecutor, capture ScreenshotCapture, annotator *ScreenshotAnnotator) *ElementSelector {
	if annotator == nil {
		annotator = NewScreenshotAnnotator(nil)
	}
	return &ElementSelector{
		executor:  executor,
		capture:   capture,
		annotator: annotator,
	}
}

// GetInteractiveElements returns all interactive elements on the page
func (es *ElementSelector) GetInteractiveElements(ctx context.Context, browserID string) ([]*Element, error) {
	// CSS selector for interactive elements
	selector := Selector{
		Type:  SelectorCSS,
		Value: "a, button, input, select, textarea, [role='button'], [onclick], [tabindex]",
	}

	return es.executor.GetElements(ctx, browserID, selector)
}

// CreateAnnotatedScreenshot creates a screenshot with annotated interactive elements
func (es *ElementSelector) CreateAnnotatedScreenshot(ctx context.Context, browserID string, opts *ScreenshotOptions) (*Screenshot, []*Element, error) {
	// Get all interactive elements
	elements, err := es.GetInteractiveElements(ctx, browserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get elements: %w", err)
	}

	// Take screenshot
	screenshot, err := es.capture.Capture(ctx, browserID, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// Annotate screenshot
	annotated, err := es.annotator.Annotate(screenshot, elements)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to annotate screenshot: %w", err)
	}

	return annotated, elements, nil
}
