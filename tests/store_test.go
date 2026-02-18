package tests

import (
	"goredis/internal/store"
	"testing"
)

func TestStoreSetAndGet(t *testing.T) {
	myStore := store.NewStore(3)
	lru := store.NewLru()

	// Test basic set and get
	myStore.Set(lru, "key1", "value1")
	val, ok := myStore.Get(lru, "key1")
	if !ok || val != "value1" {
		t.Errorf("Expected value1, got %s, ok: %v", val, ok)
	}
}

func TestStoreGetNonExistent(t *testing.T) {
	myStore := store.NewStore(3)
	lru := store.NewLru()

	// Test getting non-existent key
	val, ok := myStore.Get(lru, "nonexistent")
	if ok {
		t.Errorf("Expected ok to be false for non-existent key")
	}
	if val != "" {
		t.Errorf("Expected empty string for non-existent key, got %s", val)
	}
}

func TestStoreLRUEviction(t *testing.T) {
	myStore := store.NewStore(2)
	lru := store.NewLru()

	// Add 2 items (at capacity)
	myStore.Set(lru, "key1", "value1")
	myStore.Set(lru, "key2", "value2")

	// Add 3rd item, should evict key1 (least recently used)
	myStore.Set(lru, "key3", "value3")

	// key1 should be evicted
	_, ok := myStore.Get(lru, "key1")
	if ok {
		t.Errorf("Expected key1 to be evicted")
	}

	// key2 and key3 should still exist
	val2, ok2 := myStore.Get(lru, "key2")
	if !ok2 || val2 != "value2" {
		t.Errorf("Expected key2 to exist with value2, got %s, ok: %v", val2, ok2)
	}

	val3, ok3 := myStore.Get(lru, "key3")
	if !ok3 || val3 != "value3" {
		t.Errorf("Expected key3 to exist with value3, got %s, ok: %v", val3, ok3)
	}
}

func TestStoreLRUAccessOrder(t *testing.T) {
	myStore := store.NewStore(2)
	lru := store.NewLru()

	// Add 2 items
	myStore.Set(lru, "key1", "value1")
	myStore.Set(lru, "key2", "value2")

	// Access key1 (makes it most recently used)
	myStore.Get(lru, "key1")

	// Add key3, should evict key2 (now least recently used)
	myStore.Set(lru, "key3", "value3")

	// key2 should be evicted
	_, ok := myStore.Get(lru, "key2")
	if ok {
		t.Errorf("Expected key2 to be evicted after accessing key1")
	}

	// key1 and key3 should still exist
	val1, ok1 := myStore.Get(lru, "key1")
	if !ok1 || val1 != "value1" {
		t.Errorf("Expected key1 to exist with value1, got %s, ok: %v", val1, ok1)
	}

	val3, ok3 := myStore.Get(lru, "key3")
	if !ok3 || val3 != "value3" {
		t.Errorf("Expected key3 to exist with value3, got %s, ok: %v", val3, ok3)
	}
}

func TestStoreCapacityOne(t *testing.T) {
	myStore := store.NewStore(1)
	lru := store.NewLru()

	// Add first item
	myStore.Set(lru, "key1", "value1")
	val1, ok1 := myStore.Get(lru, "key1")
	if !ok1 || val1 != "value1" {
		t.Errorf("Expected key1 to exist, got ok: %v", ok1)
	}

	// Add second item, should evict first
	myStore.Set(lru, "key2", "value2")
	_, ok := myStore.Get(lru, "key1")
	if ok {
		t.Errorf("Expected key1 to be evicted")
	}

	val2, ok2 := myStore.Get(lru, "key2")
	if !ok2 || val2 != "value2" {
		t.Errorf("Expected key2 to exist with value2, got %s, ok: %v", val2, ok2)
	}

	// Add third item, should evict second
	myStore.Set(lru, "key3", "value3")
	_, ok = myStore.Get(lru, "key2")
	if ok {
		t.Errorf("Expected key2 to be evicted")
	}

	val3, ok3 := myStore.Get(lru, "key3")
	if !ok3 || val3 != "value3" {
		t.Errorf("Expected key3 to exist with value3, got %s, ok: %v", val3, ok3)
	}
}

func TestStoreMultipleAccesses(t *testing.T) {
	myStore := store.NewStore(3)
	lru := store.NewLru()

	// Add 3 items
	myStore.Set(lru, "key1", "value1")
	myStore.Set(lru, "key2", "value2")
	myStore.Set(lru, "key3", "value3")

	// Access key1 multiple times
	myStore.Get(lru, "key1")
	myStore.Get(lru, "key1")
	myStore.Get(lru, "key1")

	// Add key4, should evict key2 (oldest without access)
	myStore.Set(lru, "key4", "value4")

	// key2 should be evicted
	_, ok := myStore.Get(lru, "key2")
	if ok {
		t.Errorf("Expected key2 to be evicted")
	}

	// Others should exist
	_, ok1 := myStore.Get(lru, "key1")
	_, ok3 := myStore.Get(lru, "key3")
	_, ok4 := myStore.Get(lru, "key4")

	if !ok1 || !ok3 || !ok4 {
		t.Errorf("Expected key1, key3, key4 to exist")
	}
}

func TestStoreDelete(t *testing.T) {
	myStore := store.NewStore(3)
	lru := store.NewLru()

	myStore.Set(lru, "key1", "value1")

	// Test successful delete
	deleted := myStore.Delete("key1")
	if !deleted {
		t.Errorf("Expected Delete to return true")
	}

	// Verify key is gone
	_, ok := myStore.Get(lru, "key1")
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
	lru := store.NewLru()

	// Set initial value
	myStore.Set(lru, "key1", "value1")
	myStore.Set(lru, "key2", "value2")

	// Update key1 with new value
	myStore.Set(lru, "key1", "updated_value1")

	// Should have updated value
	val, ok := myStore.Get(lru, "key1")
	if !ok || val != "updated_value1" {
		t.Errorf("Expected updated_value1, got %s, ok: %v", val, ok)
	}

	// Both keys should still exist
	val2, ok2 := myStore.Get(lru, "key2")
	if !ok2 || val2 != "value2" {
		t.Errorf("Expected key2 to exist with value2")
	}
}
