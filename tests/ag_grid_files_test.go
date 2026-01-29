package tests

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

func TestAgGridFilesystem(t *testing.T) {
	// Build
	tmpBin, err := os.CreateTemp("", "sqliter-test-bin-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpBin.Close()
	defer os.Remove(tmpBin.Name())

	// Build command. We assume we are in the module root or tests dir.
	// Try to detect root.
	wd, _ := os.Getwd()
	buildArgs := []string{"build", "-o", tmpBin.Name(), "./cmd/sqliter"}
	if strings.HasSuffix(wd, "tests") {
		buildArgs = []string{"build", "-o", tmpBin.Name(), "../cmd/sqliter"}
	}

	buildCmd := exec.Command("go", buildArgs...)
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, out)
	}

	// Launch
	// Arg: sample_data
	arg := "sample_data"
	if strings.HasSuffix(wd, "tests") {
		arg = "../sample_data"
	}

	serverCmd := exec.Command(tmpBin.Name(), arg)
	stdout, err := serverCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
		}
	}()

	// Wait for URL
	urlChan := make(chan string)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			// Log checking
			// fmt.Println(line)
			if strings.Contains(line, "Listening at:") {
				parts := strings.Split(line, "Listening at: ")
				if len(parts) > 1 {
					url := strings.TrimSpace(parts[1])
					// Ensure it uses valid localhost for chromedp if [::]:0 was used
					// The server prints http://[::1]:PORT or similar. Chrome handles [::1] usually.
					urlChan <- url
				}
			}
		}
	}()

	var serverURL string
	select {
	case u := <-urlChan:
		serverURL = u
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for server startup")
	}

	t.Logf("Connecting to %s", serverURL)

	// Chromedp
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var nodes []*cdp.Node
	// We wait for typical AgGrid classes
	err = chromedp.Run(ctx,
		chromedp.Navigate(serverURL),
		chromedp.WaitVisible(".ag-root", chromedp.ByQuery),                          // Wait for grid root
		chromedp.WaitVisible(".ag-center-cols-container .ag-row", chromedp.ByQuery), // Wait for at least one row
		chromedp.Nodes(".ag-center-cols-container .ag-row", &nodes, chromedp.ByQueryAll),
	)
	if err != nil {
		t.Fatalf("Chromedp error: %v", err)
	}

	rowCount := len(nodes)
	t.Logf("Found %d rows", rowCount)

	if rowCount <= 5 {
		t.Fatalf("Expected more than 5 rows, got %d", rowCount)
	}
}
