// Spike: Validate backwards scanning approach for finding sync point
//
// Key questions:
// 1. Can we efficiently find a UUID in a JSONL file?
// 2. Is reading the file forward once better than backwards byte scanning?
// 3. What's the memory cost of various approaches?
//
// Run: go run spike_backwards_scan.go -file <path-to-jsonl>

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

type Entry struct {
	UUID string `json:"uuid"`
	Type string `json:"type"`
}

func main() {
	file := flag.String("file", "", "JSONL file to scan")
	targetUUID := flag.String("uuid", "", "UUID to find (empty = use last importable)")
	flag.Parse()

	if *file == "" {
		fmt.Println("Usage: go run spike_backwards_scan.go -file <path>")
		os.Exit(1)
	}

	// First pass: collect all importable UUIDs (simulates what we'd track)
	fmt.Println("=== Collecting all importable entries ===")
	start := time.Now()
	uuids, err := collectImportableUUIDs(*file)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d importable entries in %v\n\n", len(uuids), time.Since(start))

	if len(uuids) == 0 {
		fmt.Println("No importable entries found")
		os.Exit(1)
	}

	// Use last UUID as target if not specified
	target := *targetUUID
	if target == "" {
		target = uuids[len(uuids)-1]
		fmt.Printf("Using last importable UUID as target: %s\n\n", target)
	}

	// Approach 1: Forward scan with early termination
	fmt.Println("=== Approach 1: Forward scan ===")
	start = time.Now()
	idx1, entriesAfter1, err := forwardScan(*file, target)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Found at index %d, %d entries after, took %v\n\n", idx1, entriesAfter1, time.Since(start))
	}

	// Approach 2: Read all, find index, slice
	fmt.Println("=== Approach 2: Load all, find index ===")
	start = time.Now()
	idx2, entriesAfter2, err := loadAllFindIndex(*file, target)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Found at index %d, %d entries after, took %v\n\n", idx2, entriesAfter2, time.Since(start))
	}

	// Test: Find a mid-file UUID
	if len(uuids) > 10 {
		midTarget := uuids[len(uuids)/2]
		fmt.Printf("=== Testing mid-file UUID: %s ===\n", midTarget)

		start = time.Now()
		idx, after, _ := forwardScan(*file, midTarget)
		fmt.Printf("Forward scan: index %d, %d after, %v\n", idx, after, time.Since(start))

		start = time.Now()
		idx, after, _ = loadAllFindIndex(*file, midTarget)
		fmt.Printf("Load all: index %d, %d after, %v\n\n", idx, after, time.Since(start))
	}

	// Approach 3: Build UUID->line index during parse (single pass, get new entries)
	fmt.Println("=== Approach 3: Single-pass with UUID lookup ===")
	start = time.Now()
	newEntries, err := singlePassFromUUID(*file, target)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Found %d entries after target, took %v\n", len(newEntries), time.Since(start))
	}
}

// collectImportableUUIDs reads file and returns all UUIDs for user/assistant entries
func collectImportableUUIDs(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var uuids []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024) // 10MB buffer

	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if e.UUID != "" && (e.Type == "user" || e.Type == "assistant") {
			uuids = append(uuids, e.UUID)
		}
	}
	return uuids, scanner.Err()
}

// forwardScan reads file line by line, returns index of target and count of entries after
func forwardScan(path, targetUUID string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return -1, 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)

	foundIdx := -1
	idx := 0
	importableCount := 0

	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if e.UUID == "" || (e.Type != "user" && e.Type != "assistant") {
			continue
		}

		if foundIdx == -1 && e.UUID == targetUUID {
			foundIdx = idx
		} else if foundIdx >= 0 {
			importableCount++
		}
		idx++
	}

	return foundIdx, importableCount, scanner.Err()
}

// loadAllFindIndex loads all entries, finds target, returns entries after
func loadAllFindIndex(path, targetUUID string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return -1, 0, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)

	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if e.UUID != "" && (e.Type == "user" || e.Type == "assistant") {
			entries = append(entries, e)
		}
	}
	if err := scanner.Err(); err != nil {
		return -1, 0, err
	}

	for i, e := range entries {
		if e.UUID == targetUUID {
			return i, len(entries) - i - 1, nil
		}
	}
	return -1, 0, fmt.Errorf("UUID not found")
}

// singlePassFromUUID reads file, returns entries after target UUID
func singlePassFromUUID(path, targetUUID string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)

	var result []Entry
	found := false

	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if e.UUID == "" || (e.Type != "user" && e.Type != "assistant") {
			continue
		}

		if found {
			result = append(result, e)
		} else if e.UUID == targetUUID {
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("UUID not found")
	}
	return result, scanner.Err()
}
