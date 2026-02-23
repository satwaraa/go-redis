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
	count := 0

	for node := str.lru.Head; node != nil; node = node.next {
		if node.isExpired() {
			continue
		}
		snapshot.Entries = append(snapshot.Entries, SnapshotEntry{

			Key:      node.key,
			Value:    node.value,
			ExpireAt: node.expireAt,
		})
		count++
	}

	backup, readErr := os.ReadFile(filepath)
	hasExistingFile := readErr == nil

	if readErr != nil && !os.IsNotExist(readErr) {
		return fmt.Errorf("read file failed: %w", readErr)
	}

	// If existing file, wipe it before writing to avoid partial override
	if hasExistingFile {
		wipeOldDataError := os.WriteFile(filepath, []byte{}, 0644)
		if wipeOldDataError != nil {
			writeBackupError := os.WriteFile(filepath, backup, 0644)
			if writeBackupError != nil {
				return fmt.Errorf("wipe old data failed: %w\n Restored old backup", wipeOldDataError)
			}
			return fmt.Errorf("wipe old data failed: %w\n Restored old backup", wipeOldDataError)
		}
	}

	data, marshalErr := json.MarshalIndent(snapshot, "", " ")
	if marshalErr != nil {
		return fmt.Errorf("marshal failed: %w", marshalErr)
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
	str.lru.Head = nil
	str.lru.Tail = nil

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

		str.lru.AddToTail(node)
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
				fmt.Printf("\nAuto-save failed: %v\n", err)
			} else {
				fmt.Printf("\nAuto-save complete (%s)\n", filepath)
			}
			fmt.Print("goredis> ")
		}
	}()
}

// SaveOnShutdown saves before program exits and returns a channel
func (s *Store) SaveOnShutdown(filepath string) <-chan struct{} {
	done := make(chan struct{})
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
		close(done)
		os.Exit(0)
	}()

	return done
}
