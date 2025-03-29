package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/copyleftdev/goscry/internal/dom"
)

func main() {
	// Configure logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Starting ChromeDP verification test...")

	// Create Chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("mute-audio", true),
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
	log.Println("Running ChromeDP verification action...")
	err := chromedp.Run(ctx, dom.VerifyChromedpWorkingAction(&result))

	// Check results
	if err != nil {
		log.Printf("❌ ChromeDP test FAILED: %v", err)
		os.Exit(1)
	} else {
		log.Printf("✅ ChromeDP test PASSED with results:")
		for k, v := range result {
			fmt.Printf("  - %s: %v\n", k, v)
		}
		os.Exit(0)
	}
}
