# Spike Results: Backwards Scanning vs Forward Scan

**Date**: 2026-01-19
**Spike**: `spike_backwards_scan.go`

## Questions Tested

1. Can we efficiently find a UUID in a JSONL file?
2. Is reading the file forward once better than backwards byte scanning?
3. What's the memory cost of various approaches?

## Test Files

| File Size | Line Count | Importable Entries |
|-----------|------------|-------------------|
| 14MB | 838 | 551 |
| 48MB | 415 | 88 |

Note: Large file has fewer importable entries but larger messages (likely with tool use blocks).

## Approaches Tested

### Approach 1: Forward Scan with Index Tracking
Read file line-by-line, track index, count entries after target.

### Approach 2: Load All, Find Index
Load all entries into slice, find target, calculate remaining.

### Approach 3: Single-Pass with UUID Lookup
Read file line-by-line, start collecting after target UUID found.

## Results

### 14MB File (551 importable entries)

| Approach | Time | Notes |
|----------|------|-------|
| Forward scan | 57ms | Found at index 550 |
| Load all | 58ms | Same result |
| Single-pass | 57ms | Returns entries directly |

### 48MB File (88 importable entries)

| Approach | Time | Notes |
|----------|------|-------|
| Forward scan | 198ms | Found at index 87 |
| Load all | 190ms | Same result |
| Single-pass | 191ms | Returns entries directly |

## Key Findings

1. **All approaches perform similarly** - JSON parsing is the bottleneck, not search strategy
2. **Forward scan is sufficient** - No need for complex backwards byte scanning
3. **Single-pass is simplest** - Read once, return new entries after sync point
4. **Memory is bounded** - bufio scanner with 10MB buffer handles all files

## Recommendation

**Use forward single-pass approach:**

```go
func ParseFileFromUUID(path, lastKnownUUID string) ([]HistoryEntry, error) {
    // If no lastKnownUUID, return all importable entries (fresh import)
    // If UUID found, return entries after that point
    // If UUID not found, return error (divergence detected)
}
```

**No backwards scanning needed** - The complexity is not justified:
- Files are append-only, so new content is always at the end
- We need to parse to filter importable entries anyway
- Forward scan is O(n) regardless of sync point location
- Single file read is fast enough (200ms worst case)

## Implications for Implementation

1. **mtime check is the big win** - Skip unchanged files entirely (no parsing)
2. **Changed file handling is simple** - Single forward pass, return new entries
3. **Divergence detection** - If UUID not found, mark gap and import all
4. **No memory concerns** - Current approach with scanner is fine

## Files

- `spike_backwards_scan.go` - Test implementation
