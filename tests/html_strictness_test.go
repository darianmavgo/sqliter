package tests

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestHTMLStrictness(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Scanning repository for HTML compliance at: %s\n", root)

	// User defined allowed list + implied structural/table/form tags
	allowedTags := map[string]bool{
		"table":    true,
		"thead":    true,
		"tbody":    true,
		"tfoot":    true,
		"tr":       true,
		"td":       true,
		"th":       true,
		"form":     true,
		"input":    true,
		"button":   true,
		"select":   true,
		"textarea": true,
		"label":    true,
		"option":   true,
		"fieldset": true,
		"legend":   true,
		"datalist": true,
		"optgroup": true,
		"a":        true,
	}

	// Structural tags allowed outside/as body
	structuralTags := map[string]bool{
		"html":     true,
		"head":     true,
		"body":     true,
		"meta":     true,
		"link":     true,
		"title":    true,
		"script":   true,
		"style":    true,
		"!doctype": true,
	}

	// Matches <tagName but not <!--. Specifically allows !doctype for HTML files.
	tagRegex := regexp.MustCompile(`(?i)<([a-z][a-z0-9]*|!doctype)`)

	foundViolations := false

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" || info.Name() == "test_output" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".html" {
			return nil
		}

		// Skip this test file
		if strings.HasSuffix(path, "html_strictness_test.go") {
			return nil
		}

		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		matches := tagRegex.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			tagName := strings.ToLower(match[1])
			if allowedTags[tagName] || structuralTags[tagName] {
				continue
			}

			// Violation found (e.g. div, h1, ul, li)
			relPath, _ := filepath.Rel(root, path)
			fmt.Printf("VIOLATION: Found disallowed tag <%s> in %s\n", tagName, relPath)
			foundViolations = true
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	if foundViolations {
		t.Errorf("HTML Strictness Test FAILED: Non-table/form elements found in the codebase. See log for details.")
	} else {
		fmt.Println("HTML Strictness Test PASSED: All tags are compliant.")
	}

}
