package store

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// SnapshotEntry represents a single key-value pair with metadata
type SnapshotEntry struct {
	Key      string     `json:"key"`
	Value    string     `json:"value"`
	ExpireAt *time.Time `json:"expire_at,omitempty"`
}

// Snapshot represents entire store state
type Snapshot struct {
	Version  string          `json:"version"`
	Capacity int             `json:"capacity"`
	Entries  []SnapshotEntry `json:"entries"`
}

func (str *Store) SaveSnapshot(filepath string) error {
	str.mu.Lock()
	defer str.mu.Unlock()
	snapshot := Snapshot{
		Version:  "1.0",
		Capacity: str.capacity,
		Entries:  make([]SnapshotEntry, 0, len(str.data)),
	}

	for key, node := range str.data {

		// skip expired key.
		if node.isExpired() {
			continue
		}
		entry := SnapshotEntry{
			Key:      key,
			Value:    node.value,
			ExpireAt: node.expireAt,
		}
		snapshot.Entries = append(snapshot.Entries, entry)
		//marshal to json
	}
	data, error := json.MarshalIndent(snapshot, "", " ")
	if error != nil {
		return fmt.Errorf("marshal failed: %w", error)
	}
	// Write to file
	err := os.WriteFile(filepath, data, 0644)
	if err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}

	return nil

}

func (str *Store) LoadSnapshot(filepath string) error {
	// Check if file exists
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No snapshot, start fresh
		}
		return fmt.Errorf("read file failed: %w", err)
	}

	// Unmarshal JSON
	var snapshot Snapshot
	err = json.Unmarshal(data, &snapshot)
	if err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	str.mu.Lock()
	defer str.mu.Unlock()

	// Load entries (skip expired)
	now := time.Now()
	for _, entry := range snapshot.Entries {
		// Skip if expired
		if entry.ExpireAt != nil && now.After(*entry.ExpireAt) {
			continue
		}

		node := &Node{
			key:      entry.Key,
			value:    entry.Value,
			expireAt: entry.ExpireAt,
		}

		str.lru.AddToHead(node)
		str.data[entry.Key] = node

		// Stop if capacity reached
		if len(str.data) >= str.capacity {
			break
		}
	}

	return nil
}

// EnableAutoSave starts background goroutine to save periodically
func (str *Store) EnableAutoSave(filepath string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			err := str.SaveSnapshot(filepath)
			if err != nil {
				// Log error (add proper logging in GOR-32)
				fmt.Printf("Auto-save failed: %v\n", err)
			}
		}
	}()
}

// SaveOnShutdown saves before program exits
func (s *Store) SaveOnShutdown(filepath string) {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down, saving data...")
		err := s.SaveSnapshot(filepath)
		if err != nil {
			fmt.Printf("Save failed: %v\n", err)
		} else {
			fmt.Println("Data saved successfully")
		}
		os.Exit(0)
	}()
}
