package dom

import (
	"context"
	"os"
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

	// Detect if running in CI environment (GitHub Actions sets this env var)
	isCI := os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"
	if isCI {
		t.Log("Running in CI environment - adjusting Chrome settings")
	}

	// Create Chrome options for testing - with additional settings for CI
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

	// Add CI-specific options
	if isCI {
		opts = append(opts,
			chromedp.Flag("disable-background-networking", true),
			chromedp.Flag("disable-background-timer-throttling", true),
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("disable-ipc-flooding-protection", true),
			chromedp.Flag("enable-automation", true),
		)
	}

	// Create allocator context with timeout
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAllocator()

	// Create Chrome browser context
	ctx, cancelBrowser := chromedp.NewContext(allocatorCtx, 
		chromedp.WithLogf(t.Logf), // Add logging to help with debugging
	)
	defer cancelBrowser()

	// Set timeout for the entire test - longer for CI environments
	timeout := 30 * time.Second
	if isCI {
		timeout = 60 * time.Second // Increase timeout for CI environments
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Log Chrome startup
	t.Log("Starting Chrome instance...")

	// Initialize result map to store test outcomes
	result := make(map[string]interface{})

	// Implement retry mechanism for better reliability
	var err error
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		t.Logf("ChromeDP test attempt %d of %d", attempt, maxRetries)
		
		// Run the verification action
		err = chromedp.Run(ctx, chromedp.Navigate("about:blank")) // First navigate to a blank page as a warmup
		if err != nil {
			t.Logf("Warmup navigation failed: %v", err)
			if attempt < maxRetries {
				time.Sleep(2 * time.Second) // Wait before retry
				continue
			}
			t.Fatalf("ChromeDP initialization failed after %d attempts: %v", maxRetries, err)
		}
		
		// Now run the actual test
		err = chromedp.Run(ctx, VerifyChromedpWorkingAction(&result))
		
		// Check for errors
		if err == nil {
			break // Success, exit the loop
		}
		
		t.Logf("ChromeDP test attempt %d failed: %v", attempt, err)
		if attempt < maxRetries {
			time.Sleep(2 * time.Second) // Wait before retry
		}
	}

	// If we still have an error after all retries
	if err != nil {
		t.Fatalf("ChromeDP test failed after %d attempts: %v", maxRetries, err)
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
