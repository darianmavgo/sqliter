# SQLiter Test Plan

## Executive Summary

This document provides a comprehensive assessment of the current test coverage in the sqliter codebase and recommendations for improving test quality and coverage.

**Current Coverage Statistics:**
- `sqliter` package: **50.5%** coverage
- `common` package: **0%** coverage
- `server` package: **0%** coverage
- `cmd/servelocal`: **0%** coverage
- `cmd/demomemory`: **0%** coverage

## Current Test Inventory

### Unit Tests (`sqliter/` package)

#### `config_test.go`
**Coverage:** Configuration loading and defaults
**Tests:**
- `TestDefaultConfig` - Validates default configuration values
- `TestLoadConfig` - Tests HCL config file parsing
- `TestLoadConfigDefaults` - Tests fallback to defaults when config is empty

**Strengths:**
- Good coverage of configuration basics
- Tests both file-based and default configurations

**Gaps:**
- No tests for invalid HCL syntax
- Missing tests for partial config files
- No validation of new fields: `AutoSelectTb0`, `RowCRUD`, `LogDir`, `Verbose`

#### `html_table_test.go`
**Coverage:** HTML table generation
**Tests:**
- `TestStartHTMLTable` - Tests table header generation
- `TestWriteHTMLRow` - Tests row generation
- `TestEndHTMLTable` - Tests table closing tags

**Strengths:**
- Covers basic HTML generation
- Validates embedded CSS injection

**Gaps:**
- No tests for `TableWriter` struct and its methods
- Missing tests for `EnableEditable()` and `SetStickyHeader()` methods
- No tests for template execution errors
- No tests for different title scenarios
- No validation of editable mode header injection

### Integration Tests (`tests/` package)

#### `row_crud_test.go`
**Coverage:** CRUD operations via HTTP
**Tests:**
- `TestRowCRUD` - Tests CREATE, UPDATE, DELETE operations

**Strengths:**
- End-to-end testing of CRUD functionality
- Validates database state after operations

**Gaps:**
- No tests for CRUD with `RowCRUD=false`
- Missing error cases (invalid JSON, SQL injection attempts)
- No tests for NULL value handling
- No tests for concurrent CRUD operations

#### `links_browser_test.go`
**Coverage:** Browser navigation and auto-redirect
**Tests:**
- `TestBrowserLinksFlow` - Tests database browsing and auto-redirect to single table

**Strengths:**
- Tests realistic user flow
- Validates auto-redirect feature

**Gaps:**
- No tests for multi-table databases
- No tests with `AutoRedirectSingleTable=false`
- Missing tests for non-existent tables/databases

#### `html_strictness_test.go`
**Coverage:** HTML tag compliance
**Tests:**
- `TestHTMLStrictness` - Enforces table/form-only HTML policy

**Strengths:**
- Enforces architectural constraint
- Scans entire codebase

**Gaps:**
- Could be extended to validate accessibility attributes

#### `ux_structure_test.go`
**Coverage:** HTML structure requirements
**Tests:**
- `TestTableUXStructure` - Validates edit bar, row IDs, and UX elements

**Strengths:**
- Tests specific UI requirements
- Validates editable mode structure

**Gaps:**
- No tests for sticky header behavior
- Missing tests for tb0 special handling
- No tests with editable mode disabled

#### `clear_non_default_css_test.go`
**Coverage:** CSS file policy
**Tests:**
- `TestOnlyDefaultCSS` - Ensures only default.css exists

**Strengths:**
- Enforces single CSS file policy
- Scans entire repository

## Critical Coverage Gaps

### 1. `server` Package (0% Coverage) - **HIGH PRIORITY**

The server package implements core HTTP routing and request handling but has **zero test coverage**.

**Missing Tests:**
- ❌ `ServeHTTP` routing logic
- ❌ URL parsing with banquet library
- ❌ Database file listing (`listFiles`)
- ❌ Table listing (`listTables`)
- ❌ Query execution (`queryTable`)
- ❌ CRUD request handling (`handleCRUD`)
- ❌ Security: directory traversal prevention
- ❌ Error handling and logging (`logError`)
- ❌ tb0 special handling (editable=false, sticky=false)
- ❌ Title derivation from DataSetPath

**Recommended Tests:**

```go
// server/server_test.go
func TestServeHTTP_ListFiles(t *testing.T)
func TestServeHTTP_ListTables(t *testing.T)
func TestServeHTTP_QueryTable(t *testing.T)
func TestServeHTTP_DirectoryTraversal(t *testing.T)
func TestServeHTTP_NonExistentDB(t *testing.T)
func TestServeHTTP_Tb0SpecialHandling(t *testing.T)
func TestServeHTTP_TitleFromDataSetPath(t *testing.T)
func TestHandleCRUD_InvalidJSON(t *testing.T)
func TestHandleCRUD_DisabledCRUD(t *testing.T)
```

### 2. `common` Package (0% Coverage) - **HIGH PRIORITY**

The common package handles SQL query construction and is vulnerable to SQL injection.

**Missing Tests:**
- ❌ `ConstructSQL` with various banquet parameters
- ❌ `ConstructInsert` edge cases
- ❌ `ConstructUpdate` with NULL values
- ❌ `ConstructDelete` with empty where clauses
- ❌ SQL injection vulnerability testing

**Recommended Tests:**

```go
// common/sql_test.go
func TestConstructSQL_BasicQuery(t *testing.T)
func TestConstructSQL_WithWhere(t *testing.T)
func TestConstructSQL_WithOrderBy(t *testing.T)
func TestConstructSQL_WithLimit(t *testing.T)
func TestConstructInsert_MultipleColumns(t *testing.T)
func TestConstructUpdate_NullValues(t *testing.T)
func TestConstructDelete_NullWhere(t *testing.T)
```

### 3. `cmd` Packages (0% Coverage) - **MEDIUM PRIORITY**

Command-line entry points have no tests.

**Missing Tests:**
- ❌ `servelocal/main.go` - Server startup and configuration
- ❌ `demomemory/main.go` - In-memory demo functionality

**Recommended Tests:**

```go
// cmd/servelocal/main_test.go
func TestServerStartup(t *testing.T)
func TestConfigLoading(t *testing.T)
func TestPortBinding(t *testing.T)
```

### 4. New Features (Untested) - **HIGH PRIORITY**

Recent features added without corresponding tests:

**tb0 Special Handling:**
- ❌ Verify editable mode is disabled for tb0
- ❌ Verify sticky header is disabled for tb0
- ❌ Test title derivation from DataSetPath

**Per-Request TableWriter:**
- ❌ Verify thread-safety of per-request TableWriter creation
- ❌ Test independent settings for different tables

## Recommendations

### Phase 1: Critical Gaps (Week 1-2)

1. **Add `server` package tests** - Focus on HTTP routing and error handling
   - Priority: Test security features (directory traversal, invalid paths)
   - Test new tb0 functionality
   - Test title derivation logic

2. **Add `common` package tests** - Focus on SQL construction
   - Priority: Test edge cases and potential SQL injection vectors
   - Test NULL value handling
   - Test empty/missing parameters

3. **Add tests for new features**
   - Test tb0 special handling end-to-end
   - Test DataSetPath to title conversion
   - Test per-request TableWriter settings

### Phase 2: Enhanced Coverage (Week 3-4)

4. **Expand `sqliter` package tests**
   - Test `TableWriter` methods (`SetStickyHeader`, `EnableEditable`)
   - Test template execution error handling
   - Test embedded asset loading failures

5. **Add error scenario tests**
   - Database connection failures
   - Malformed requests
   - Concurrent access patterns

6. **Add integration tests**
   - Multi-table database browsing
   - Complex SQL queries via banquet URLs
   - File upload/download flows (if applicable)

### Phase 3: Quality & Edge Cases (Week 5+)

7. **Add performance tests**
   - Large table rendering
   - Concurrent request handling
   - Memory usage patterns

8. **Add security tests**
   - SQL injection attempts
   - XSS prevention
   - CSRF protection (if applicable)

9. **Add accessibility tests**
   - Screen reader compatibility
   - Keyboard navigation
   - ARIA attributes

## Testing Standards

### Unit Test Requirements
- Use table-driven tests where applicable
- Test both success and error paths
- Mock external dependencies (filesystem, database)
- Aim for >80% coverage per package

### Integration Test Requirements
- Use `testutil.GetTestOutputDir()` for temporary files
- Clean up resources with `defer`
- Test realistic user flows
- Validate both HTTP responses and database state

### Test Naming Convention
```go
func Test<FunctionName>_<Scenario>(t *testing.T)
// Examples:
func TestServeHTTP_ValidRequest(t *testing.T)
func TestConstructSQL_WithNullWhere(t *testing.T)
```

## Running Tests

### All Tests with Coverage
```bash
go test -cover ./...
```

### Specific Package
```bash
go test -v -cover ./server
go test -v -cover ./common
go test -v -cover ./sqliter
```

### With Coverage Report
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests Only
```bash
go test -v ./tests
```

## Success Metrics

### Coverage Targets (3-Month Goal)
- `server` package: **0% → 70%+**
- `common` package: **0% → 80%+**
- `sqliter` package: **50% → 80%+**
- Overall project: **~15% → 60%+**

### Quality Targets
- ✅ All critical paths tested
- ✅ All new features include tests
- ✅ No regressions in existing functionality
- ✅ Security vulnerabilities tested and prevented

## Next Steps

1. Review this test plan with the team
2. Prioritize Phase 1 tests
3. Create test implementation tasks
4. Set up CI/CD to enforce coverage minimums
5. Integrate coverage reporting into PR reviews
