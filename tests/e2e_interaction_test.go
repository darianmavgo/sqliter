package tests

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// TestE2EInteractions verifies AG Grid interactions in the full app.
// Requires: Chrome/Chromium installed.
func TestE2EInteractions(t *testing.T) {
	// 1. Build Server
	tmpBin, err := os.CreateTemp("", "sqliter-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpBin.Close()
	defer os.Remove(tmpBin.Name())

	// Detect path to cmd/sqliter
	wd, _ := os.Getwd()
	buildTarget := "./cmd/sqliter"
	if strings.HasSuffix(wd, "tests") {
		buildTarget = "../cmd/sqliter"
	}

	// Build
	out, err := exec.Command("go", "build", "-o", tmpBin.Name(), buildTarget).CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, out)
	}

	// 2. Prepare Data
	dataDir := "sample_data"
	if strings.HasSuffix(wd, "tests") {
		dataDir = "../sample_data"
	}

	// Start Server
	serverCmd := exec.Command(tmpBin.Name(), dataDir)
	stdout, _ := serverCmd.StdoutPipe()
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
		}
	}()

	// 3. Wait for Startup
	urlChan := make(chan string)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Listening at:") {
				parts := strings.Split(line, "Listening at: ")
				if len(parts) > 1 {
					urlChan <- strings.TrimSpace(parts[1])
				}
			}
		}
	}()

	var baseURL string
	select {
	case u := <-urlChan:
		baseURL = u
	case <-time.After(5 * time.Second):
		t.Fatal("Server startup timeout")
	}
	t.Logf("Server running at %s", baseURL)

	// 4. Setup Chromedp WITHOUT aggressive timeout on context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.WindowSize(1280, 1024),
	)
	allocCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAllocator()

	ctx, cancelContext := chromedp.NewContext(allocCtx)
	defer cancelContext() // This closes the tab/browser when test ends

	// Helper for Screenshot
	takeScreenshot := func(name string) {
		var buf []byte
		if err := chromedp.Run(ctx, chromedp.CaptureScreenshot(&buf)); err == nil && len(buf) > 0 {
			os.WriteFile(name, buf, 0644)
			t.Logf("Saved screenshot to %s", name)
		} else {
			t.Logf("Failed to capture screenshot: %v", err)
		}
		var url, title string
		chromedp.Run(ctx, chromedp.Location(&url), chromedp.Title(&title))
		t.Logf("Current State - URL: %s, Title: %s", url, title)
	}

	// 5. Run Scenarios with manual timeout control
	runErr := make(chan error, 1)

	go func() {
		t.Log("Navigating to home...")
		if err := chromedp.Run(ctx, chromedp.Navigate(baseURL)); err != nil {
			runErr <- err
			return
		}

		// Wait for ROOT of standard file browser or grid
		if err := chromedp.Run(ctx, chromedp.WaitVisible(".ag-root")); err != nil {
			runErr <- err
			return
		}

		t.Log("Looking for database files...")
		// Click database
		err = chromedp.Run(ctx,
			chromedp.WaitVisible(`a[href$=".db"], a[href$=".sqlite"]`, chromedp.ByQuery),
			chromedp.Click(`a[href$=".db"], a[href$=".sqlite"]`, chromedp.ByQuery),
		)
		if err == nil {
			t.Log("Clicked a database file.")
			time.Sleep(2 * time.Second)
		} else {
			t.Logf("No database links found: %v", err)
		}

		// Check for Drilldown links (Table List) or if we are already in Grid View
		// Wait for AgGrid headers to appear
		if err := chromedp.Run(ctx, chromedp.WaitVisible(".ag-header-cell-text", chromedp.ByQuery)); err != nil {
			runErr <- err
			return
		}

		// Check if we are in Table List by looking for "Table Name" header
		var tableListMode bool
		err = chromedp.Run(ctx,
			chromedp.Evaluate(`Array.from(document.querySelectorAll('.ag-header-cell-text')).some(el => el.innerText === "Table Name")`, &tableListMode),
		)
		if err != nil {
			t.Logf("Failed to evaluate table mode: %v", err)
		}

		if tableListMode {
			t.Log("Detected Table List mode. Clicking table...")
			err = chromedp.Run(ctx,
				chromedp.WaitVisible(`a[href*="/"]`, chromedp.ByQuery),
				chromedp.Click(`a[href*="/"]`, chromedp.ByQuery),
			)
			if err != nil {
				runErr <- err
				return
			}
			time.Sleep(2 * time.Second)
		} else {
			t.Log("Detected Grid View (auto-redirect?). Skipping table selection.")
		}

		// Try to sort
		t.Log("Attempting to Sort...")
		var firstRowTextBefore string
		var firstRowTextAfter string

		err = chromedp.Run(ctx,
			chromedp.WaitVisible(".ag-header-cell-label", chromedp.ByQuery),
			chromedp.Text(`.ag-center-cols-container .ag-row[row-index="0"] .ag-cell`, &firstRowTextBefore, chromedp.ByQuery),
			chromedp.Click(`.ag-header-cell-label`, chromedp.ByQuery),
			chromedp.Sleep(2*time.Second),
			chromedp.Text(`.ag-center-cols-container .ag-row[row-index="0"] .ag-cell`, &firstRowTextAfter, chromedp.ByQuery),
		)
		if err != nil {
			runErr <- err
			return
		}

		t.Logf("Row before sort: %q", firstRowTextBefore)
		t.Logf("Row after sort: %q", firstRowTextAfter)

		runErr <- nil // Success
	}()

	select {
	case err := <-runErr:
		if err != nil {
			takeScreenshot("e2e_error.png")
			t.Fatalf("E2E Scenario failed: %v", err)
		}
	case <-time.After(60 * time.Second):
		takeScreenshot("e2e_timeout.png")
		t.Fatal("E2E Scenario timed out")
	}
}
