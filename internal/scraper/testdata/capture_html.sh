#!/bin/bash

# Script to capture fresh HTML from Michelin Guide and replace test files
# Usage: ./capture_html.sh

echo "Capturing fresh Michelin Guide HTML for testing..."

USER_AGENT="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

# Backup existing files
echo "Creating backups..."
[ -f restaurant_detail.html ] && cp restaurant_detail.html restaurant_detail.html.backup
[ -f restaurant_list.html ] && cp restaurant_list.html restaurant_list.html.backup

# Capture restaurant detail page and replace test file
echo "Capturing restaurant detail page..."
curl -H "User-Agent: $USER_AGENT" \
    -s "https://guide.michelin.com/en/singapore-region/singapore/restaurant/les-amis" \
    >restaurant_detail.html

if [ $? -eq 0 ] && [ -s restaurant_detail.html ]; then
    echo "✅ Restaurant detail page captured and replaced"
    echo "   File: restaurant_detail.html"
    echo "   Size: $(wc -c <restaurant_detail.html) bytes"
else
    echo "❌ Failed to capture restaurant detail page"
    [ -f restaurant_detail.html.backup ] && mv restaurant_detail.html.backup restaurant_detail.html
fi

# Capture restaurant list page and replace test file
echo "Capturing restaurant list page..."
curl -H "User-Agent: $USER_AGENT" \
    -s "https://guide.michelin.com/en/restaurants/3-stars-michelin" \
    >restaurant_list.html

if [ $? -eq 0 ] && [ -s restaurant_list.html ]; then
    echo "✅ Restaurant list page captured and replaced"
    echo "   File: restaurant_list.html"
    echo "   Size: $(wc -c <restaurant_list.html) bytes"
else
    echo "❌ Failed to capture restaurant list page"
    [ -f restaurant_list.html.backup ] && mv restaurant_list.html.backup restaurant_list.html
fi

# Clean up successful backups
[ -f restaurant_detail.html ] && [ -s restaurant_detail.html ] && rm -f restaurant_detail.html.backup
[ -f restaurant_list.html ] && [ -s restaurant_list.html ] && rm -f restaurant_list.html.backup

echo ""
echo "🚀 Next steps:"
echo "1. Inspect the updated HTML files to understand data structure"
echo "2. Update test expectations in scraper_test.go if needed"
echo "3. Run tests: go test -v"
echo "4. Tests now use fresh real data from Michelin Guide!"
