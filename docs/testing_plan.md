# Automated Testing Strategy for SQLiter

## Goal
Replace manual verification of UI interactions (filtering, sorting) and URL parsing with a comprehensive automated test suite. The strategy prioritizes **Integration** and **End-to-End (E2E)** tests to mirror user behavior.

## User Review Required
> [!NOTE]
> We will leverage the existing `chromedp` setup for E2E tests, which requires Chrome/Chromium to be installed on the machine running tests.

## Proposed Changes

### 1. Backend API Integration Tests (Go)
**Focus**: Validate that API parameters correctly translate to SQL queries and JSON results. This is faster/more reliable for logic verification than full browser tests.
- **Location**: `sqliter/api_test.go` (or `tests/api_test.go`)
- **Scope**:
    - **Sorting**: Call `/sqliter/rows?db=...&table=...&sortCol=name&sortDir=desc`. Verify first row returned.
    - **Filtering**: Call `/sqliter/rows` with `filterModel`. Verify count.
    - **Banquet URLs**: Call `/sqliter/rows?path=/db.sqlite/table:limit=5`. Verify strict obedience to the URL string.
    - **Pagination**: Verify `start`/`end` params slice the data correctly.

### 2. End-to-End UI Interaction Tests (Go + Chromedp)
**Focus**: Verify the "glue" between React, AG Grid, and the Backend. Simulate actual user clicks.
- **Location**: `tests/e2e_interaction_test.go`
- **Scope**:
    - **Sort Click**:
        1. Load grid.
        2. Click "Name" header.
        3. Wait for grid update.
        4. Read first row text to confirm sort.
    - **Filter Input**:
        1. Open filter menu for a column.
        2. Type "TargetValue".
        3. Wait for grid update.
        4. Verify row count matches expectation.
    - **URL Navigation**:
        1. Navigate directly to `/Index.sqlite/tb0`.
        2. Verify grid loads.
        3. Navigate to a deep link (if supported).

### 3. Banquet Compliance Suite
**Focus**: ensure deep Banquet URL syntax support.
- **Location**: `tests/banquet_compliance_test.go`
- **Scope**:
    - Table-driven test iterate through complex URL patterns:
        - `/db/table:sort=col`
        - `/db/table:sort=col:desc`
        - `/db/table:a=1` (where clauses)
    - Assert that the *Compose* SQL query aligns with expectations (requires exposing query info in API, which we did via `"sql"` field in response).

## Verification Plan
### Automated Tests
Run all tests with a single command:
```bash
go test ./... -v
```

### Manual Verification
- None required once suite is green.
