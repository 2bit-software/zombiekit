# Search API Contract

**Date**: 2025-12-22
**Feature**: 011-webgui-search

## Endpoint: GET /search

Searches all plugins implementing the Searchable interface and returns aggregated results.

### Request

**Method**: GET
**Path**: /search
**Content-Type**: Not applicable (query parameters only)

#### Query Parameters

| Parameter | Type   | Required | Description                    |
|-----------|--------|----------|--------------------------------|
| q         | string | Yes      | Search query text              |

### Response (HTML Partial)

**Content-Type**: text/html
**Status**: 200 OK

Returns an HTML fragment suitable for insertion into the search results dropdown.

#### Response Structure

```html
<!-- When results found -->
<div class="search-results">
  {{range .Results}}
  {{if .Items}}
  <div class="search-group">
    <div class="search-group-header">{{.PluginName}}</div>
    {{range .Items}}
    <a href="{{.URL}}"
       hx-get="{{.URL}}"
       hx-target="#content"
       hx-push-url="true"
       data-search-result
       class="search-result">
      {{.Title}}
    </a>
    {{end}}
  </div>
  {{end}}
  {{end}}
</div>

<!-- When no results -->
<div class="search-no-results">
  No results found for "{{.Query}}"
</div>

<!-- When query empty -->
<!-- Empty response (no content) -->
```

### Behavior

1. **Empty Query**: Returns empty response (no HTML)
2. **No Matches**: Returns "No results found" message
3. **Plugin Error**: Logs error, continues with other plugins
4. **Multiple Plugins**: Results grouped by plugin, ordered by registration

### Example

**Request**:
```
GET /search?q=config HTTP/1.1
HX-Request: true
```

**Response**:
```html
<div class="search-results">
  <div class="search-group">
    <div class="search-group-header">Memory</div>
    <a href="/memory/app-config" hx-get="/memory/app-config" hx-target="#content" hx-push-url="true" data-search-result class="search-result">app-config</a>
    <a href="/memory/db-config" hx-get="/memory/db-config" hx-target="#content" hx-push-url="true" data-search-result class="search-result">db-config</a>
  </div>
</div>
```

## Error Responses

| Status | Condition              | Response                    |
|--------|------------------------|-----------------------------|
| 200    | Always                 | HTML fragment (may be empty)|
| 500    | Template render failure| "Internal Server Error"     |

Note: Search errors from individual plugins do not fail the request. The failing plugin simply returns no results.
