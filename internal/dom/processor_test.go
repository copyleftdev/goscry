package dom

import (
	"context"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// TestChromedpWorks tests if chromedp works properly by interacting with a real website
func TestChromedpWorks(t *testing.T) {
	// Skip test if running in short mode (-short flag)
	if testing.Short() {
		t.Skip("Skipping chromedp test in short mode")
	}

	// Create Chrome options for testing
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("mute-audio", true),
		chromedp.WindowSize(1280, 1024),
		chromedp.IgnoreCertErrors,
	)

	// Create allocator context with timeout
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAllocator()

	// Create Chrome browser context with timeout
	ctx, cancelBrowser := chromedp.NewContext(allocatorCtx)
	defer cancelBrowser()

	// Set timeout for the entire test
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Initialize result map to store test outcomes
	result := make(map[string]interface{})

	// Run the verification action
	err := chromedp.Run(ctx, VerifyChromedpWorkingAction(&result))

	// Check for errors
	if err != nil {
		t.Fatalf("ChromeDP test failed: %v", err)
	}

	// Verify expected results
	t.Logf("ChromeDP test results: %+v", result)

	// Check title
	if title, ok := result["title"].(string); !ok || title == "" {
		t.Error("Failed to get page title")
	} else {
		t.Logf("Page title: %s", title)
	}

	// Check element presence
	if isPresent, ok := result["element_present"].(bool); !ok || !isPresent {
		t.Error("Failed to detect H1 element")
	}

	// Check HTML was received
	if htmlLength, ok := result["html_length"].(int); !ok || htmlLength < 100 {
		t.Error("HTML content seems invalid or too short")
	}

	// Check screenshot was taken
	if screenshotSize, ok := result["screenshot_size"].(int); !ok || screenshotSize < 1000 {
		t.Error("Screenshot seems invalid or too small")
	}
}
