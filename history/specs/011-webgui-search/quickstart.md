# Quickstart: Web GUI Search Bar

**Date**: 2025-12-22
**Feature**: 011-webgui-search

## Overview

This feature adds a search bar to the web GUI header that searches across all plugins implementing the `Searchable` interface. Results appear in a dropdown with up to 3 items per plugin.

## Prerequisites

- Go 1.24+
- Running web server (`brains gui` command)
- At least one plugin implementing `Searchable` (memory plugin does)

## Usage

### Basic Search

1. Open the web GUI at `http://localhost:8080`
2. Click the search bar in the header (or press `/` to focus)
3. Type your search query
4. Results appear after 300ms pause in typing
5. Click a result or use arrow keys + Enter to navigate

### Keyboard Shortcuts

| Key        | Action                          |
|------------|---------------------------------|
| `/`        | Focus search bar                |
| Arrow Down | Move to next result             |
| Arrow Up   | Move to previous result         |
| Enter      | Navigate to selected result     |
| Escape     | Close dropdown / clear search   |

## For Plugin Developers

To make your plugin searchable:

1. Import the search package:
   ```go
   import "github.com/2bit-software/zombiekit/internal/search"
   ```

2. Implement the `Searchable` interface on your plugin:
   ```go
   func (p *Plugin) Search(query string, maxResults int, sortOrder search.SortOrder) ([]search.SearchResult, error) {
       // Return matching items
       return []search.SearchResult{
           {Title: "My Item", URL: "/myplugin/my-item"},
       }, nil
   }
   ```

3. The search bar will automatically include your plugin's results.

### SearchResult Fields

| Field | Description                                          |
|-------|------------------------------------------------------|
| Title | Display name shown in dropdown                       |
| URL   | Relative path for navigation (e.g., "/memory/notes") |

### Sorting Options

The `sortOrder` parameter supports:
- `search.SortRelevance` (default)
- `search.SortCreatedDate`
- `search.SortUpdatedDate`
- `search.SortLastUsed`
- `search.SortName`

The search bar always uses `SortRelevance`.

## Testing

Run the web server and test manually:

```bash
# Start the server
brains gui

# In another terminal, test the search endpoint
curl "http://localhost:8080/search?q=test"
```

## Troubleshooting

### No results appearing
- Verify your plugin implements `Searchable`
- Check that items exist matching your query
- Look for errors in server logs

### Results not updating
- Wait 300ms after typing (debounce)
- Try pressing Escape and searching again
- Check browser console for JavaScript errors

### Dropdown not closing
- Click outside the search area
- Press Escape key
- Navigate to a result (closes automatically)
