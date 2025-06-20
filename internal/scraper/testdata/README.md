# Integration Test Data & Workflow

This directory contains HTML fixtures captured from real Michelin Guide pages for testing your scraper.

## Test Files

-   `restaurant_detail.html` - Real restaurant detail page HTML
-   `restaurant_list.html` - Real restaurant list page HTML
-   `capture_html.sh` - Script to capture fresh HTML

## Workflow for Updates

### Step 1: Capture Fresh HTML

```bash
cd internal/scraper/testdata
./capture_html.sh
```

### Step 2: Run Tests to Check Status

```bash
cd internal/scraper
go test -v
```

### Step 3: If Tests Fail (Website Changes)

1. **Inspect captured HTML** to see what changed
2. **Update XPath selectors** in `internal/scraper/xpath.go` if needed
3. **Update test expectations** in `scraper_test.go` if data structure changed
4. **Re-run tests** until they pass

## Maintenance Checklist

When Michelin Guide website changes:

-   [ ] Capture fresh HTML: `./capture_html.sh`
-   [ ] Run tests: `go test -v`
-   [ ] If failing, check XPath selectors in `xpath.go`
-   [ ] Update test expectations in `scraper_test.go`
-   [ ] Verify data extraction still works correctly
-   [ ] All tests passing
