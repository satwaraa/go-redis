package tests

import (
	"fmt"
	"goredis/internal/store"
	"sync"
	"testing"
)

func TestStoreSetAndGet(t *testing.T) {
	myStore := store.NewStore(3)

	// Test basic set and get
	myStore.Set("key1", "value1")
	val, ok := myStore.Get("key1")
	if !ok || val != "value1" {
		t.Errorf("Expected value1, got %s, ok: %v", val, ok)
	}
}

func TestStoreGetNonExistent(t *testing.T) {
	myStore := store.NewStore(3)
	// Test getting non-existent key
	val, ok := myStore.Get("nonexistent")
	if ok {
		t.Errorf("Expected ok to be false for non-existent key")
	}
	if val != "" {
		t.Errorf("Expected empty string for non-existent key, got %s", val)
	}
}
func TestStoreLRUEviction(t *testing.T) {
	myStore := store.NewStore(2)

	// Add 2 items (at capacity)
	myStore.Set("key1", "value1")
	myStore.Set("key2", "value2")

	// Add 3rd item, should evict key1 (least recently used)
	myStore.Set("key3", "value3")

	// key1 should be evicted
	_, ok := myStore.Get("key1")
	if ok {
		t.Errorf("Expected key1 to be evicted")
	}

	// key2 and key3 should still exist
	val2, ok2 := myStore.Get("key2")
	if !ok2 || val2 != "value2" {
		t.Errorf("Expected key2 to exist with value2, got %s, ok: %v", val2, ok2)
	}

	val3, ok3 := myStore.Get("key3")
	if !ok3 || val3 != "value3" {
		t.Errorf("Expected key3 to exist with value3, got %s, ok: %v", val3, ok3)
	}
}

func TestStoreLRUAccessOrder(t *testing.T) {
	myStore := store.NewStore(2)
	// Add 2 items
	myStore.Set("key1", "value1")
	myStore.Set("key2", "value2")

	// Access key1 (makes it most recently used)
	myStore.Get("key1")

	// Add key3, should evict key2 (now least recently used)
	myStore.Set("key3", "value3")

	// key2 should be evicted
	_, ok := myStore.Get("key2")
	if ok {
		t.Errorf("Expected key2 to be evicted after accessing key1")
	}

	// key1 and key3 should still exist
	val1, ok1 := myStore.Get("key1")
	if !ok1 || val1 != "value1" {
		t.Errorf("Expected key1 to exist with value1, got %s, ok: %v", val1, ok1)
	}

	val3, ok3 := myStore.Get("key3")
	if !ok3 || val3 != "value3" {
		t.Errorf("Expected key3 to exist with value3, got %s, ok: %v", val3, ok3)
	}
}

func TestStoreCapacityOne(t *testing.T) {
	myStore := store.NewStore(1)

	// Add first item
	myStore.Set("key1", "value1")
	val1, ok1 := myStore.Get("key1")
	if !ok1 || val1 != "value1" {
		t.Errorf("Expected key1 to exist, got ok: %v", ok1)
	}

	// Add second item, should evict first
	myStore.Set("key2", "value2")
	_, ok := myStore.Get("key1")
	if ok {
		t.Errorf("Expected key1 to be evicted")
	}

	val2, ok2 := myStore.Get("key2")
	if !ok2 || val2 != "value2" {
		t.Errorf("Expected key2 to exist with value2, got %s, ok: %v", val2, ok2)
	}

	// Add third item, should evict second
	myStore.Set("key3", "value3")
	_, ok = myStore.Get("key2")
	if ok {
		t.Errorf("Expected key2 to be evicted")
	}

	val3, ok3 := myStore.Get("key3")
	if !ok3 || val3 != "value3" {
		t.Errorf("Expected key3 to exist with value3, got %s, ok: %v", val3, ok3)
	}
}

func TestStoreMultipleAccesses(t *testing.T) {
	myStore := store.NewStore(3)

	// Add 3 items
	myStore.Set("key1", "value1")
	myStore.Set("key2", "value2")
	myStore.Set("key3", "value3")

	// Access key1 multiple times
	myStore.Get("key1")
	myStore.Get("key1")
	myStore.Get("key1")

	// Add key4, should evict key2 (oldest without access)
	myStore.Set("key4", "value4")

	// key2 should be evicted
	_, ok := myStore.Get("key2")
	if ok {
		t.Errorf("Expected key2 to be evicted")
	}

	// Others should exist
	_, ok1 := myStore.Get("key1")
	_, ok3 := myStore.Get("key3")
	_, ok4 := myStore.Get("key4")

	if !ok1 || !ok3 || !ok4 {
		t.Errorf("Expected key1, key3, key4 to exist")
	}
}

func TestStoreDelete(t *testing.T) {
	myStore := store.NewStore(3)

	myStore.Set("key1", "value1")

	// Test successful delete
	deleted := myStore.Delete("key1")
	if !deleted {
		t.Errorf("Expected Delete to return true")
	}

	// Verify key is gone
	_, ok := myStore.Get("key1")
	if ok {
		t.Errorf("Expected key1 to be deleted")
	}

	// Test deleting non-existent key
	deleted = myStore.Delete("nonexistent")
	if deleted {
		t.Errorf("Expected Delete to return false for non-existent key")
	}
}

func TestStoreUpdateExistingKey(t *testing.T) {
	myStore := store.NewStore(2)

	// Set initial value
	myStore.Set("key1", "value1")
	myStore.Set("key2", "value2")

	// Update key1 with new value
	myStore.Set("key1", "updated_value1")

	// Should have updated value
	val, ok := myStore.Get("key1")
	if !ok || val != "updated_value1" {
		t.Errorf("Expected updated_value1, got %s, ok: %v", val, ok)
	}

	// Both keys should still exist
	val2, ok2 := myStore.Get("key2")
	if !ok2 || val2 != "value2" {
		t.Errorf("Expected key2 to exist with value2")
	}
}

func TestUpdateDoesNotCreateDuplicateNodes(t *testing.T) {
	myStore := store.NewStore(2)

	myStore.Set("key1", "value1")
	myStore.Set("key2", "value2")

	// Update key1 multiple times — should NOT increase store size
	myStore.Set("key1", "v1_update1")
	myStore.Set("key1", "v1_update2")
	myStore.Set("key1", "v1_update3")

	// Both keys should still exist (no eviction from updates)
	val1, ok1 := myStore.Get("key1")
	if !ok1 || val1 != "v1_update3" {
		t.Errorf("Expected v1_update3, got %s, ok: %v", val1, ok1)
	}

	val2, ok2 := myStore.Get("key2")
	if !ok2 || val2 != "value2" {
		t.Errorf("Expected key2 to still exist with value2, got %s, ok: %v", val2, ok2)
	}
}

func TestUpdateMovesToHead(t *testing.T) {
	myStore := store.NewStore(2)

	myStore.Set("old", "value_old")
	myStore.Set("new", "value_new")

	// Update "old" — moves it to head, making "new" the tail (LRU)
	myStore.Set("old", "value_old_updated")

	// Add a third key — should evict "new" (now LRU), not "old"
	myStore.Set("key3", "value3")

	// "old" should exist (was moved to head by update)
	val, ok := myStore.Get("old")
	if !ok || val != "value_old_updated" {
		t.Errorf("Expected old to exist with value_old_updated, got %s, ok: %v", val, ok)
	}

	// "new" should be evicted
	_, ok2 := myStore.Get("new")
	if ok2 {
		t.Errorf("Expected 'new' to be evicted since 'old' was moved to head by update")
	}

	// key3 should exist
	val3, ok3 := myStore.Get("key3")
	if !ok3 || val3 != "value3" {
		t.Errorf("Expected key3 to exist with value3, got %s, ok: %v", val3, ok3)
	}
}

func TestUpdateAtCapacityDoesNotEvict(t *testing.T) {
	myStore := store.NewStore(2)

	myStore.Set("key1", "value1")
	myStore.Set("key2", "value2")

	// Store is at capacity. Updating existing key should NOT evict anything
	myStore.Set("key2", "updated_value2")

	val1, ok1 := myStore.Get("key1")
	if !ok1 || val1 != "value1" {
		t.Errorf("Expected key1 to still exist after update, got %s, ok: %v", val1, ok1)
	}

	val2, ok2 := myStore.Get("key2")
	if !ok2 || val2 != "updated_value2" {
		t.Errorf("Expected updated_value2, got %s, ok: %v", val2, ok2)
	}
}

func TestConcurrentAccess(t *testing.T) {
	myStore := store.NewStore(100)
	var wg sync.WaitGroup

	// Launch 100 goroutines writing
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			defer wg.Done()
			myStore.Set(fmt.Sprintf("key%d", idx), "value")
		}(i)
	}

	// Launch 100 goroutines reading
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			defer wg.Done()
			myStore.Get(fmt.Sprintf("key%d", idx))
		}(i)
	}

	wg.Wait()
	// Should not crash with race conditions!
}
