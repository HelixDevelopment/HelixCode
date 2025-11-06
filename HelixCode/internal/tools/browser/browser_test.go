package browser

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChromeDiscovery tests Chrome discovery functionality
func TestChromeDiscovery(t *testing.T) {
	discovery := NewDefaultChromeDiscovery()

	t.Run("find chrome", func(t *testing.T) {
		path, err := discovery.FindChrome()
		if err != nil {
			t.Skip("Chrome not installed, skipping test")
		}
		assert.NotEmpty(t, path)

		// Verify the file exists
		_, statErr := os.Stat(path)
		assert.NoError(t, statErr, "Chrome executable should exist")
	})

	t.Run("get default paths", func(t *testing.T) {
		paths := discovery.GetDefaultPaths()
		assert.NotEmpty(t, paths, "Should return default paths for platform")
	})

	t.Run("find chrome version", func(t *testing.T) {
		path, err := discovery.FindChrome()
		if err != nil {
			t.Skip("Chrome not installed, skipping test")
		}

		version, err := discovery.FindChromeVersion(path)
		if err != nil {
			t.Logf("Could not determine Chrome version: %v", err)
		} else {
			assert.NotEmpty(t, version, "Version should not be empty")
			t.Logf("Chrome version: %s", version)
		}
	})

	t.Run("find all chrome installations", func(t *testing.T) {
		chromes, err := discovery.FindAll()
		if err != nil {
			t.Skip("No Chrome installations found, skipping test")
		}

		assert.NotEmpty(t, chromes, "Should find at least one Chrome installation")
		for _, chrome := range chromes {
			t.Logf("Found %s at %s (version: %s)", chrome.Type, chrome.Path, chrome.Version)
		}
	})

	t.Run("get preferred chrome", func(t *testing.T) {
		path, err := GetPreferredChrome()
		if err != nil {
			t.Skip("No Chrome installation found, skipping test")
		}
		assert.NotEmpty(t, path)
		t.Logf("Preferred Chrome: %s", path)
	})

	t.Run("chrome type string", func(t *testing.T) {
		assert.Equal(t, "Chrome", ChromeTypeChrome.String())
		assert.Equal(t, "Chromium", ChromeTypeChromium.String())
		assert.Equal(t, "Edge", ChromeTypeEdge.String())
		assert.Equal(t, "Brave", ChromeTypeBrave.String())
	})
}

// TestBrowserLaunch tests browser launch functionality
func TestBrowserLaunch(t *testing.T) {
	discovery := NewDefaultChromeDiscovery()
	controller := NewDefaultController(discovery)

	t.Run("launch headless browser", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		opts := &LaunchOptions{
			Headless: true,
			Width:    1280,
			Height:   720,
		}

		browser, err := controller.Launch(ctx, opts)
		if err != nil {
			t.Skipf("Failed to launch browser: %v", err)
			return
		}
		defer controller.Close(browser.ID)

		assert.NotEmpty(t, browser.ID)
		// Note: WSEndpoint may be empty in newer chromedp versions
		t.Logf("Launched browser %s", browser.ID)
	})

	t.Run("launch with default options", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		browser, err := controller.Launch(ctx, nil)
		if err != nil {
			t.Skipf("Failed to launch browser: %v", err)
			return
		}
		defer controller.Close(browser.ID)

		assert.NotEmpty(t, browser.ID)
	})

	t.Run("list browsers", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		browser, err := controller.Launch(ctx, DefaultLaunchOptions())
		if err != nil {
			t.Skipf("Failed to launch browser: %v", err)
			return
		}
		defer controller.Close(browser.ID)

		browsers := controller.ListBrowsers()
		assert.NotEmpty(t, browsers, "Should have at least one browser")

		found := false
		for _, b := range browsers {
			if b.ID == browser.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Launched browser should be in list")
	})

	t.Run("get browser by id", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		browser, err := controller.Launch(ctx, DefaultLaunchOptions())
		if err != nil {
			t.Skipf("Failed to launch browser: %v", err)
			return
		}
		defer controller.Close(browser.ID)

		retrieved, err := controller.GetBrowser(browser.ID)
		require.NoError(t, err)
		assert.Equal(t, browser.ID, retrieved.ID)
	})

	t.Run("close browser", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		browser, err := controller.Launch(ctx, DefaultLaunchOptions())
		if err != nil {
			t.Skipf("Failed to launch browser: %v", err)
			return
		}

		err = controller.Close(browser.ID)
		assert.NoError(t, err)

		_, err = controller.GetBrowser(browser.ID)
		assert.Error(t, err, "Browser should not be found after closing")
	})
}

// TestBrowserActions tests browser action execution
func TestBrowserActions(t *testing.T) {
	discovery := NewDefaultChromeDiscovery()
	controller := NewDefaultController(discovery)
	executor := NewDefaultActionExecutor(controller)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	browser, err := controller.Launch(ctx, DefaultLaunchOptions())
	if err != nil {
		t.Skipf("Failed to launch browser: %v", err)
		return
	}
	defer controller.Close(browser.ID)

	t.Run("navigate to url", func(t *testing.T) {
		err := executor.Navigate(ctx, browser.ID, "https://example.com")
		assert.NoError(t, err)

		// Wait a bit for navigation to complete
		time.Sleep(2 * time.Second)

		url, title, err := executor.GetPageInfo(ctx, browser.ID)
		assert.NoError(t, err)
		assert.Contains(t, url, "example.com")
		assert.NotEmpty(t, title)
		t.Logf("Navigated to %s - %s", url, title)
	})

	t.Run("get page info", func(t *testing.T) {
		err := executor.Navigate(ctx, browser.ID, "https://example.com")
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		url, title, err := executor.GetPageInfo(ctx, browser.ID)
		assert.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.NotEmpty(t, title)
	})

	t.Run("evaluate javascript", func(t *testing.T) {
		err := executor.Navigate(ctx, browser.ID, "https://example.com")
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		result, err := executor.Evaluate(ctx, browser.ID, "document.title")
		assert.NoError(t, err)
		assert.NotNil(t, result.Value)
		t.Logf("JavaScript result: %v (type: %s)", result.Value, result.Type)
	})

	t.Run("get element", func(t *testing.T) {
		err := executor.Navigate(ctx, browser.ID, "https://example.com")
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		element, err := executor.GetElement(ctx, browser.ID, Selector{
			Type:  SelectorCSS,
			Value: "h1",
		})
		if err == nil {
			assert.NotNil(t, element)
			assert.Equal(t, "h1", element.TagName)
			t.Logf("Found element: %s with text: %s", element.TagName, element.Text)
		}
	})

	t.Run("get multiple elements", func(t *testing.T) {
		err := executor.Navigate(ctx, browser.ID, "https://example.com")
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		elements, err := executor.GetElements(ctx, browser.ID, Selector{
			Type:  SelectorCSS,
			Value: "p",
		})
		if err == nil {
			t.Logf("Found %d paragraph elements", len(elements))
		}
	})

	t.Run("scroll page", func(t *testing.T) {
		err := executor.Navigate(ctx, browser.ID, "https://example.com")
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		err = executor.Scroll(ctx, browser.ID, &ScrollOptions{
			X: 0,
			Y: 500,
		})
		assert.NoError(t, err)
	})
}

// TestScreenshot tests screenshot capture
func TestScreenshot(t *testing.T) {
	discovery := NewDefaultChromeDiscovery()
	controller := NewDefaultController(discovery)
	executor := NewDefaultActionExecutor(controller)
	capture := NewDefaultScreenshotCapture(controller, executor)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	browser, err := controller.Launch(ctx, DefaultLaunchOptions())
	if err != nil {
		t.Skipf("Failed to launch browser: %v", err)
		return
	}
	defer controller.Close(browser.ID)

	t.Run("capture screenshot", func(t *testing.T) {
		err := executor.Navigate(ctx, browser.ID, "https://example.com")
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		screenshot, err := capture.Capture(ctx, browser.ID, &ScreenshotOptions{
			Format: FormatPNG,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, screenshot.Data)
		assert.Equal(t, FormatPNG, screenshot.Format)
		assert.Greater(t, screenshot.Width, 0)
		assert.Greater(t, screenshot.Height, 0)
		t.Logf("Captured screenshot: %dx%d, %d bytes", screenshot.Width, screenshot.Height, len(screenshot.Data))
	})

	t.Run("annotate screenshot", func(t *testing.T) {
		err := executor.Navigate(ctx, browser.ID, "https://example.com")
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		screenshot, err := capture.Capture(ctx, browser.ID, nil)
		require.NoError(t, err)

		elements, _ := executor.GetElements(ctx, browser.ID, Selector{
			Type:  SelectorCSS,
			Value: "a, button, input",
		})

		annotator := NewScreenshotAnnotator(nil)
		annotated, err := annotator.Annotate(screenshot, elements)
		assert.NoError(t, err)
		assert.NotEmpty(t, annotated.Data)
		t.Logf("Annotated screenshot with %d elements", len(elements))
	})
}

// TestConsoleMonitor tests console monitoring
func TestConsoleMonitor(t *testing.T) {
	t.Run("create console monitor", func(t *testing.T) {
		monitor := NewConsoleMonitor(nil)
		assert.NotNil(t, monitor)
		assert.NotNil(t, monitor.GetMessages())
		assert.NotNil(t, monitor.GetErrors())
	})

	t.Run("console message type string", func(t *testing.T) {
		assert.Equal(t, "log", ConsoleLog.String())
		assert.Equal(t, "info", ConsoleInfo.String())
		assert.Equal(t, "warning", ConsoleWarning.String())
		assert.Equal(t, "error", ConsoleError.String())
		assert.Equal(t, "debug", ConsoleDebug.String())
	})
}

// TestBrowserTools tests the unified BrowserTools interface
func TestBrowserTools(t *testing.T) {
	tools := NewBrowserTools(nil)
	assert.NotNil(t, tools)

	t.Run("launch browser with tools", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		browser, err := tools.LaunchBrowser(ctx, nil)
		if err != nil {
			t.Skipf("Failed to launch browser: %v", err)
			return
		}
		defer tools.CloseBrowser(browser.ID)

		assert.NotEmpty(t, browser.ID)
	})

	t.Run("navigate and screenshot", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		browser, err := tools.LaunchBrowser(ctx, nil)
		if err != nil {
			t.Skipf("Failed to launch browser: %v", err)
			return
		}
		defer tools.CloseBrowser(browser.ID)

		err = tools.Navigate(ctx, browser.ID, "https://example.com")
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		screenshot, err := tools.TakeScreenshot(ctx, browser.ID, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, screenshot.Data)
	})

	t.Run("get interactive elements", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		browser, err := tools.LaunchBrowser(ctx, nil)
		if err != nil {
			t.Skipf("Failed to launch browser: %v", err)
			return
		}
		defer tools.CloseBrowser(browser.ID)

		err = tools.Navigate(ctx, browser.ID, "https://example.com")
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		elements, err := tools.GetInteractiveElements(ctx, browser.ID)
		if err == nil {
			t.Logf("Found %d interactive elements", len(elements))
		}
	})

	t.Run("list browsers", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		browser, err := tools.LaunchBrowser(ctx, nil)
		if err != nil {
			t.Skipf("Failed to launch browser: %v", err)
			return
		}
		defer tools.CloseBrowser(browser.ID)

		browsers := tools.ListBrowsers()
		assert.NotEmpty(t, browsers)
	})
}

// TestBrowserSession tests browser session management
func TestBrowserSession(t *testing.T) {
	tools := NewBrowserTools(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	session, err := NewBrowserSession(ctx, tools, nil)
	if err != nil {
		t.Skipf("Failed to create browser session: %v", err)
		return
	}
	defer session.Close()

	t.Run("navigate in session", func(t *testing.T) {
		err := session.Navigate("https://example.com")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", session.StartURL)
		assert.Equal(t, "https://example.com", session.CurrentURL)
	})

	t.Run("take screenshot in session", func(t *testing.T) {
		err := session.Navigate("https://example.com")
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		screenshot, err := session.TakeScreenshot(nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, screenshot.Data)

		screenshots := session.GetScreenshots()
		assert.Len(t, screenshots, 1)
	})
}

// TestHelperFunctions tests helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("image format string", func(t *testing.T) {
		assert.Equal(t, "png", FormatPNG.String())
		assert.Equal(t, "jpeg", FormatJPEG.String())
		assert.Equal(t, "webp", FormatWebP.String())
	})

	t.Run("default config", func(t *testing.T) {
		config := DefaultConfig()
		assert.NotNil(t, config)
		assert.True(t, config.DefaultHeadless)
		assert.Equal(t, 1280, config.DefaultWidth)
		assert.Equal(t, 720, config.DefaultHeight)
	})

	t.Run("default launch options", func(t *testing.T) {
		opts := DefaultLaunchOptions()
		assert.NotNil(t, opts)
		assert.True(t, opts.Headless)
		assert.Equal(t, 1280, opts.Width)
		assert.Equal(t, 720, opts.Height)
	})

	t.Run("default annotation options", func(t *testing.T) {
		opts := DefaultAnnotationOptions()
		assert.NotNil(t, opts)
		assert.NotNil(t, opts.BorderColor)
		assert.NotNil(t, opts.LabelColor)
		assert.True(t, opts.ShowLabels)
		assert.True(t, opts.ShowBounds)
	})

	t.Run("default console monitor options", func(t *testing.T) {
		opts := DefaultConsoleMonitorOptions()
		assert.NotNil(t, opts)
		assert.Equal(t, 1000, opts.MaxLogSize)
		assert.True(t, opts.FilterErrors)
		assert.Equal(t, 100, opts.BufferSize)
	})
}
