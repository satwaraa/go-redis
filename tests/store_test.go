package tests

import (
	"fmt"
	"memstash/internal/store"
	"os"
	"sync"
	"testing"
	"time"
)

func TestStoreSetAndGet(t *testing.T) {
	myStore := store.NewStore(3)

	// Test basic set and get
	myStore.Set("key1", "value1")
	val, err := myStore.Get("key1")
	if err != nil || val != "value1" {
		t.Errorf("Expected value1, got %s, err: %v", val, err)
	}
}

func TestStoreGetNonExistent(t *testing.T) {
	myStore := store.NewStore(3)
	// Test getting non-existent key
	val, err := myStore.Get("nonexistent")
	if err != store.ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound for non-existent key")
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
	_, err := myStore.Get("key1")
	if err != store.ErrKeyNotFound {
		t.Errorf("Expected key1 to be evicted")
	}

	// key2 and key3 should still exist
	val2, err2 := myStore.Get("key2")
	if err2 != nil || val2 != "value2" {
		t.Errorf("Expected key2 to exist with value2, got %s, err: %v", val2, err2)
	}

	val3, err3 := myStore.Get("key3")
	if err3 != nil || val3 != "value3" {
		t.Errorf("Expected key3 to exist with value3, got %s, err: %v", val3, err3)
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
	_, err := myStore.Get("key2")
	if err != store.ErrKeyNotFound {
		t.Errorf("Expected key2 to be evicted after accessing key1")
	}

	// key1 and key3 should still exist
	val1, err1 := myStore.Get("key1")
	if err1 != nil || val1 != "value1" {
		t.Errorf("Expected key1 to exist with value1, got %s, err: %v", val1, err1)
	}

	val3, err3 := myStore.Get("key3")
	if err3 != nil || val3 != "value3" {
		t.Errorf("Expected key3 to exist with value3, got %s, err: %v", val3, err3)
	}
}

func TestStoreCapacityOne(t *testing.T) {
	myStore := store.NewStore(1)

	// Add first item
	myStore.Set("key1", "value1")
	val1, err1 := myStore.Get("key1")
	if err1 != nil || val1 != "value1" {
		t.Errorf("Expected key1 to exist, got err: %v", err1)
	}

	// Add second item, should evict first
	myStore.Set("key2", "value2")
	_, err := myStore.Get("key1")
	if err != store.ErrKeyNotFound {
		t.Errorf("Expected key1 to be evicted")
	}

	val2, err2 := myStore.Get("key2")
	if err2 != nil || val2 != "value2" {
		t.Errorf("Expected key2 to exist with value2, got %s, err: %v", val2, err2)
	}

	// Add third item, should evict second
	myStore.Set("key3", "value3")
	_, err = myStore.Get("key2")
	if err != store.ErrKeyNotFound {
		t.Errorf("Expected key2 to be evicted")
	}

	val3, err3 := myStore.Get("key3")
	if err3 != nil || val3 != "value3" {
		t.Errorf("Expected key3 to exist with value3, got %s, err: %v", val3, err3)
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
	_, err := myStore.Get("key2")
	if err != store.ErrKeyNotFound {
		t.Errorf("Expected key2 to be evicted")
	}

	// Others should exist
	_, err1 := myStore.Get("key1")
	_, err3 := myStore.Get("key3")
	_, err4 := myStore.Get("key4")

	if err1 != nil || err3 != nil || err4 != nil {
		t.Errorf("Expected key1, key3, key4 to exist")
	}
}

func TestStoreDelete(t *testing.T) {
	myStore := store.NewStore(3)

	myStore.Set("key1", "value1")

	// Test successful delete
	err := myStore.Delete("key1")
	if err != nil {
		t.Errorf("Expected Delete to succeed, got err: %v", err)
	}

	// Verify key is gone
	_, err = myStore.Get("key1")
	if err != store.ErrKeyNotFound {
		t.Errorf("Expected key1 to be deleted")
	}

	// Test deleting non-existent key
	err = myStore.Delete("nonexistent")
	if err != store.ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound for non-existent key, got: %v", err)
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
	val, err := myStore.Get("key1")
	if err != nil || val != "updated_value1" {
		t.Errorf("Expected updated_value1, got %s, err: %v", val, err)
	}

	// Both keys should still exist
	val2, err2 := myStore.Get("key2")
	if err2 != nil || val2 != "value2" {
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
	val1, err1 := myStore.Get("key1")
	if err1 != nil || val1 != "v1_update3" {
		t.Errorf("Expected v1_update3, got %s, err: %v", val1, err1)
	}

	val2, err2 := myStore.Get("key2")
	if err2 != nil || val2 != "value2" {
		t.Errorf("Expected key2 to still exist with value2, got %s, err: %v", val2, err2)
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
	val, err := myStore.Get("old")
	if err != nil || val != "value_old_updated" {
		t.Errorf("Expected old to exist with value_old_updated, got %s, err: %v", val, err)
	}

	// "new" should be evicted
	_, err2 := myStore.Get("new")
	if err2 != store.ErrKeyNotFound {
		t.Errorf("Expected 'new' to be evicted since 'old' was moved to head by update")
	}

	// key3 should exist
	val3, err3 := myStore.Get("key3")
	if err3 != nil || val3 != "value3" {
		t.Errorf("Expected key3 to exist with value3, got %s, err: %v", val3, err3)
	}
}

func TestUpdateAtCapacityDoesNotEvict(t *testing.T) {
	myStore := store.NewStore(2)

	myStore.Set("key1", "value1")
	myStore.Set("key2", "value2")

	// Store is at capacity. Updating existing key should NOT evict anything
	myStore.Set("key2", "updated_value2")

	val1, err1 := myStore.Get("key1")
	if err1 != nil || val1 != "value1" {
		t.Errorf("Expected key1 to still exist after update, got %s, err: %v", val1, err1)
	}

	val2, err2 := myStore.Get("key2")
	if err2 != nil || val2 != "updated_value2" {
		t.Errorf("Expected updated_value2, got %s, err: %v", val2, err2)
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

func TestCleanExpiredKeysRemovesOnlyExpiredKeys(t *testing.T) {
	myStore := store.NewStore(5)

	// Set 2 keys with short TTL and 2 without TTL
	myStore.SetWithTTL("expire1", "val1", 50*time.Millisecond)
	myStore.SetWithTTL("expire2", "val2", 50*time.Millisecond)
	myStore.Set("keep1", "val_keep1")
	myStore.Set("keep2", "val_keep2")

	// Wait for TTL keys to expire
	time.Sleep(100 * time.Millisecond)

	// Trigger cleaner
	myStore.StartTTLCleaner(10 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)

	// Expired keys should be gone
	_, err1 := myStore.Get("expire1")
	if err1 != store.ErrKeyNotFound {
		t.Errorf("Expected expire1 to be cleaned, got err: %v", err1)
	}

	_, err2 := myStore.Get("expire2")
	if err2 != store.ErrKeyNotFound {
		t.Errorf("Expected expire2 to be cleaned, got err: %v", err2)
	}

	// Non-TTL keys should still exist
	val1, errK1 := myStore.Get("keep1")
	if errK1 != nil || val1 != "val_keep1" {
		t.Errorf("Expected keep1 to exist with val_keep1, got %s, err: %v", val1, errK1)
	}

	val2, errK2 := myStore.Get("keep2")
	if errK2 != nil || val2 != "val_keep2" {
		t.Errorf("Expected keep2 to exist with val_keep2, got %s, err: %v", val2, errK2)
	}
}

func TestCleanExpiredKeysEmptyStore(t *testing.T) {
	myStore := store.NewStore(5)

	// Should not panic on empty store
	myStore.StartTTLCleaner(10 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	// If we get here without a panic, test passes
}

func TestCleanExpiredKeysAllExpired(t *testing.T) {
	myStore := store.NewStore(3)

	myStore.SetWithTTL("a", "1", 50*time.Millisecond)
	myStore.SetWithTTL("b", "2", 50*time.Millisecond)
	myStore.SetWithTTL("c", "3", 50*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	myStore.StartTTLCleaner(10 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)

	// All keys should be gone
	_, errA := myStore.Get("a")
	_, errB := myStore.Get("b")
	_, errC := myStore.Get("c")

	if errA != store.ErrKeyNotFound {
		t.Errorf("Expected a to be cleaned, got err: %v", errA)
	}
	if errB != store.ErrKeyNotFound {
		t.Errorf("Expected b to be cleaned, got err: %v", errB)
	}
	if errC != store.ErrKeyNotFound {
		t.Errorf("Expected c to be cleaned, got err: %v", errC)
	}
}

func TestCleanerDoesNotRemoveNonExpiredTTLKeys(t *testing.T) {
	myStore := store.NewStore(3)

	// Set keys with long TTL — should NOT be cleaned
	myStore.SetWithTTL("alive1", "val1", 10*time.Second)
	myStore.SetWithTTL("alive2", "val2", 10*time.Second)

	myStore.StartTTLCleaner(10 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)

	val1, err1 := myStore.Get("alive1")
	if err1 != nil || val1 != "val1" {
		t.Errorf("Expected alive1 to still exist, got %s, err: %v", val1, err1)
	}

	val2, err2 := myStore.Get("alive2")
	if err2 != nil || val2 != "val2" {
		t.Errorf("Expected alive2 to still exist, got %s, err: %v", val2, err2)
	}
}

// ==================== NEW TESTS ====================

func TestExists(t *testing.T) {
	myStore := store.NewStore(3)

	myStore.Set("key1", "value1")

	if !myStore.Exists("key1") {
		t.Error("Expected key1 to exist")
	}
	if myStore.Exists("nonexistent") {
		t.Error("Expected nonexistent key to not exist")
	}

	// Delete and check again
	myStore.Delete("key1")
	if myStore.Exists("key1") {
		t.Error("Expected key1 to not exist after deletion")
	}
}

func TestExistsAfterEviction(t *testing.T) {
	myStore := store.NewStore(2)

	myStore.Set("key1", "value1")
	myStore.Set("key2", "value2")
	myStore.Set("key3", "value3") // evicts key1

	if myStore.Exists("key1") {
		t.Error("Expected key1 to not exist after eviction")
	}
	if !myStore.Exists("key2") {
		t.Error("Expected key2 to still exist")
	}
	if !myStore.Exists("key3") {
		t.Error("Expected key3 to still exist")
	}
}

func TestClear(t *testing.T) {
	myStore := store.NewStore(5)

	myStore.Set("a", "1")
	myStore.Set("b", "2")
	myStore.Set("c", "3")

	myStore.Clear()

	if myStore.Exists("a") || myStore.Exists("b") || myStore.Exists("c") {
		t.Error("Expected all keys to be cleared")
	}

	stats := myStore.Stats()
	if stats.Keys != 0 {
		t.Errorf("Expected 0 keys after clear, got %d", stats.Keys)
	}

	// Should be able to add new keys after clear
	myStore.Set("new", "value")
	val, err := myStore.Get("new")
	if err != nil || val != "value" {
		t.Errorf("Expected to set new key after clear, got %s, err: %v", val, err)
	}
}

func TestClearEmptyStore(t *testing.T) {
	myStore := store.NewStore(5)
	myStore.Clear()

	stats := myStore.Stats()
	if stats.Keys != 0 {
		t.Errorf("Expected 0 keys, got %d", stats.Keys)
	}
}

func TestKeys(t *testing.T) {
	myStore := store.NewStore(5)

	myStore.Set("alpha", "1")
	myStore.Set("beta", "2")
	myStore.Set("gamma", "3")

	keys := myStore.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}
	for _, expected := range []string{"alpha", "beta", "gamma"} {
		if !keyMap[expected] {
			t.Errorf("Expected key %s to be in Keys() output", expected)
		}
	}
}

func TestKeysEmpty(t *testing.T) {
	myStore := store.NewStore(5)
	keys := myStore.Keys()
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys, got %d", len(keys))
	}
}

func TestKeysAfterDelete(t *testing.T) {
	myStore := store.NewStore(5)

	myStore.Set("a", "1")
	myStore.Set("b", "2")
	myStore.Delete("a")

	keys := myStore.Keys()
	if len(keys) != 1 {
		t.Errorf("Expected 1 key after delete, got %d", len(keys))
	}
	if keys[0] != "b" {
		t.Errorf("Expected remaining key to be 'b', got %s", keys[0])
	}
}

func TestSetExpiry(t *testing.T) {
	myStore := store.NewStore(5)

	myStore.Set("key1", "value1")

	err := myStore.SetExpiry("key1", 100*time.Millisecond)
	if err != nil {
		t.Errorf("Expected SetExpiry to succeed, got err: %v", err)
	}

	val, err := myStore.Get("key1")
	if err != nil || val != "value1" {
		t.Errorf("Expected key1 to exist immediately after SetExpiry, got %s, err: %v", val, err)
	}

	ttl, err := myStore.GetTTL("key1")
	if err != nil {
		t.Errorf("Expected GetTTL to succeed, got err: %v", err)
	}
	if ttl <= 0 {
		t.Errorf("Expected positive TTL, got %v", ttl)
	}

	time.Sleep(150 * time.Millisecond)

	_, err = myStore.Get("key1")
	if err != store.ErrKeyExpired {
		t.Errorf("Expected ErrKeyExpired after expiry, got: %v", err)
	}
}

func TestSetExpiryOnNonExistentKey(t *testing.T) {
	myStore := store.NewStore(5)

	err := myStore.SetExpiry("nonexistent", 1*time.Second)
	if err != store.ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got: %v", err)
	}
}

func TestSetExpiryRemovesTTL(t *testing.T) {
	myStore := store.NewStore(5)

	myStore.SetWithTTL("key1", "value1", 100*time.Millisecond)

	// Remove TTL by setting <= 0
	myStore.SetExpiry("key1", -1)

	time.Sleep(150 * time.Millisecond)

	val, err := myStore.Get("key1")
	if err != nil || val != "value1" {
		t.Errorf("Expected key1 to still exist after TTL removal, got %s, err: %v", val, err)
	}

	ttl, _ := myStore.GetTTL("key1")
	if ttl != -1 {
		t.Errorf("Expected TTL -1 (no expiration), got %v", ttl)
	}
}

func TestSetWithTTLNewKey(t *testing.T) {
	myStore := store.NewStore(5)

	err := myStore.SetWithTTL("ttlkey", "ttlval", 200*time.Millisecond)
	if err != nil {
		t.Errorf("Expected SetWithTTL to succeed, got err: %v", err)
	}

	val, err := myStore.Get("ttlkey")
	if err != nil || val != "ttlval" {
		t.Errorf("Expected ttlval, got %s, err: %v", val, err)
	}

	time.Sleep(250 * time.Millisecond)

	_, err = myStore.Get("ttlkey")
	if err != store.ErrKeyExpired {
		t.Errorf("Expected ErrKeyExpired, got: %v", err)
	}
}

func TestSetWithTTLZeroDuration(t *testing.T) {
	myStore := store.NewStore(5)

	err := myStore.SetWithTTL("key", "val", 0)
	if err == nil {
		t.Error("Expected error for zero TTL")
	}
}

func TestSetEmptyKey(t *testing.T) {
	myStore := store.NewStore(5)

	err := myStore.Set("", "value")
	if err != store.ErrInvalidKey {
		t.Errorf("Expected ErrInvalidKey for empty key, got: %v", err)
	}
}

func TestGetEmptyKey(t *testing.T) {
	myStore := store.NewStore(5)

	_, err := myStore.Get("")
	if err != store.ErrInvalidKey {
		t.Errorf("Expected ErrInvalidKey for empty key, got: %v", err)
	}
}

func TestSnapshotSaveAndLoad(t *testing.T) {
	filepath := "/tmp/test_memstash_snapshot.json"

	s1 := store.NewStore(5)
	s1.Set("name", "Alice")
	s1.Set("age", "30")
	s1.Set("city", "NYC")

	err := s1.SaveSnapshot(filepath)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	s2 := store.NewStore(5)
	err = s2.LoadSnapshot(filepath)
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	for _, k := range []string{"name", "age", "city"} {
		if !s2.Exists(k) {
			t.Errorf("Expected key %s to exist after load", k)
		}
	}

	val, _ := s2.Get("name")
	if val != "Alice" {
		t.Errorf("Expected 'Alice', got %s", val)
	}
	val, _ = s2.Get("age")
	if val != "30" {
		t.Errorf("Expected '30', got %s", val)
	}
	val, _ = s2.Get("city")
	if val != "NYC" {
		t.Errorf("Expected 'NYC', got %s", val)
	}

	os.Remove(filepath)
}

func TestSnapshotSkipsExpiredKeys(t *testing.T) {
	filepath := "/tmp/test_memstash_ttl_snapshot.json"

	s1 := store.NewStore(5)
	s1.SetWithTTL("temp", "gone_soon", 50*time.Millisecond)
	s1.Set("permanent", "stays")

	time.Sleep(100 * time.Millisecond)

	err := s1.SaveSnapshot(filepath)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	s2 := store.NewStore(5)
	err = s2.LoadSnapshot(filepath)
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	if s2.Exists("temp") {
		t.Error("Expected expired key 'temp' to not be loaded")
	}

	val, err := s2.Get("permanent")
	if err != nil || val != "stays" {
		t.Errorf("Expected 'stays', got %s, err: %v", val, err)
	}

	os.Remove(filepath)
}

func TestSnapshotNonExistentFile(t *testing.T) {
	s := store.NewStore(5)
	err := s.LoadSnapshot("/tmp/nonexistent_memstash_file_12345.json")
	if err != nil {
		t.Errorf("Loading non-existent snapshot should return nil, got: %v", err)
	}
}

func TestStatsCounters(t *testing.T) {
	myStore := store.NewStore(5)

	myStore.Set("key1", "val1")
	myStore.Get("key1")        // hit
	myStore.Get("key1")        // hit
	myStore.Get("nonexistent") // miss

	stats := myStore.Stats()
	if stats.Keys != 1 {
		t.Errorf("Expected 1 key, got %d", stats.Keys)
	}
	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
	if stats.Capacity != 5 {
		t.Errorf("Expected capacity 5, got %d", stats.Capacity)
	}
}
