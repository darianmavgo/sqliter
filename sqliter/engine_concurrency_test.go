package sqliter

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrentQueryOOM(t *testing.T) {
	// Setup
	// Assuming running from repo root, sample_data is in sample_data/
	// If running from sqliter package, it's ../sample_data/
	// We'll assume repo root for now or adjust.
    // Actually, tests run in the directory of the package.

	dbPath := "../sample_data/history.db"
    absDbPath, _ := filepath.Abs(dbPath)

	cfg := &Config{
		ServeFolder: filepath.Dir(absDbPath), // Serve the folder containing the DB
	}
	engine := NewEngine(cfg)

	// Verify we can read it once
	opts := QueryOptions{
		BanquetPath: "/history.db/summary",
        Limit: 10,
	}
	_, err := engine.Query(opts)
	if err != nil {
		t.Fatalf("Initial query failed: %v", err)
	}

	// Concurrent load
	concurrency := 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

    errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()
			opts := QueryOptions{
				BanquetPath: "/history.db/summary",
                Limit: 50,
                Offset: id * 10,
			}
			_, err := engine.Query(opts)
			if err != nil {
                errors <- fmt.Errorf("Routine %d failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
    close(errors)

    failed := false
    for err := range errors {
        t.Log(err)
        failed = true
    }

    if failed {
        t.Fatal("One or more concurrent queries failed")
    }
}
