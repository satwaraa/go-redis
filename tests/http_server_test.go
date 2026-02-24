package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"memstash/internal/server"
	"memstash/internal/store"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

// helper: start an HTTP server on port 0, return server + base URL
func startTestHTTPServer(t *testing.T, capacity int) (*server.HTTPServer, string) {
	t.Helper()
	s := store.NewStore(capacity)
	srv := server.NewHTTPServer(s, 0)
	ready := srv.StartAndReady()
	<-ready
	baseURL := fmt.Sprintf("http://%s", srv.Addr().String())
	return srv, baseURL
}

// helper: decode JSON response body into a map
func decodeJSON(t *testing.T, body io.Reader) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}
	return result
}

// ==================== TESTS ====================

func TestHTTPSetAndGet(t *testing.T) {
	srv, baseURL := startTestHTTPServer(t, 10)
	defer srv.Stop()

	// SET
	body := bytes.NewBufferString(`{"value": "bar"}`)
	resp, err := http.Post(baseURL+"/keys/foo", "application/json", body)
	if err != nil {
		t.Fatalf("POST /keys/foo failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("POST /keys/foo: expected 201, got %d", resp.StatusCode)
	}

	// GET
	resp, err = http.Get(baseURL + "/keys/foo")
	if err != nil {
		t.Fatalf("GET /keys/foo failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /keys/foo: expected 200, got %d", resp.StatusCode)
	}
	data := decodeJSON(t, resp.Body)
	if data["key"] != "foo" || data["value"] != "bar" {
		t.Errorf("GET /keys/foo: expected {key:foo, value:bar}, got %v", data)
	}
}

func TestHTTPGetNonExistent(t *testing.T) {
	srv, baseURL := startTestHTTPServer(t, 10)
	defer srv.Stop()

	resp, err := http.Get(baseURL + "/keys/missing")
	if err != nil {
		t.Fatalf("GET /keys/missing failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET /keys/missing: expected 404, got %d", resp.StatusCode)
	}
}

func TestHTTPDeleteKey(t *testing.T) {
	srv, baseURL := startTestHTTPServer(t, 10)
	defer srv.Stop()

	// Set a key first
	body := bytes.NewBufferString(`{"value": "val"}`)
	http.Post(baseURL+"/keys/delme", "application/json", body)

	// DELETE existing
	req, _ := http.NewRequest("DELETE", baseURL+"/keys/delme", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /keys/delme failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("DELETE existing: expected 200, got %d", resp.StatusCode)
	}

	// DELETE non-existent
	req, _ = http.NewRequest("DELETE", baseURL+"/keys/delme", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /keys/delme again failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("DELETE non-existent: expected 404, got %d", resp.StatusCode)
	}

	// GET after DELETE
	resp, err = http.Get(baseURL + "/keys/delme")
	if err != nil {
		t.Fatalf("GET /keys/delme after delete failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET after DELETE: expected 404, got %d", resp.StatusCode)
	}
}

func TestHTTPListKeys(t *testing.T) {
	srv, baseURL := startTestHTTPServer(t, 10)
	defer srv.Stop()

	// Empty store
	resp, err := http.Get(baseURL + "/keys")
	if err != nil {
		t.Fatalf("GET /keys failed: %v", err)
	}
	defer resp.Body.Close()
	data := decodeJSON(t, resp.Body)
	keys := data["keys"].([]any)
	if len(keys) != 0 {
		t.Errorf("GET /keys empty: expected 0 keys, got %d", len(keys))
	}

	// Add some keys
	for _, k := range []string{"a", "b", "c"} {
		body := bytes.NewBufferString(fmt.Sprintf(`{"value": "%s_val"}`, k))
		http.Post(baseURL+"/keys/"+k, "application/json", body)
	}

	resp, err = http.Get(baseURL + "/keys")
	if err != nil {
		t.Fatalf("GET /keys failed: %v", err)
	}
	defer resp.Body.Close()
	data = decodeJSON(t, resp.Body)
	keys = data["keys"].([]any)
	if len(keys) != 3 {
		t.Errorf("GET /keys: expected 3 keys, got %d", len(keys))
	}
	count := data["count"].(float64)
	if int(count) != 3 {
		t.Errorf("GET /keys: expected count=3, got %v", count)
	}
}

func TestHTTPStats(t *testing.T) {
	srv, baseURL := startTestHTTPServer(t, 10)
	defer srv.Stop()

	// Set and get to generate hits/misses
	body := bytes.NewBufferString(`{"value": "v"}`)
	http.Post(baseURL+"/keys/k", "application/json", body)
	http.Get(baseURL + "/keys/k")       // hit
	http.Get(baseURL + "/keys/missing") // miss

	resp, err := http.Get(baseURL + "/stats")
	if err != nil {
		t.Fatalf("GET /stats failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /stats: expected 200, got %d", resp.StatusCode)
	}
	data := decodeJSON(t, resp.Body)

	if data["capacity"].(float64) != 10 {
		t.Errorf("Expected capacity=10, got %v", data["capacity"])
	}
	if data["keys"].(float64) != 1 {
		t.Errorf("Expected keys=1, got %v", data["keys"])
	}
	if data["hits"].(float64) < 1 {
		t.Errorf("Expected hits>=1, got %v", data["hits"])
	}
	if data["misses"].(float64) < 1 {
		t.Errorf("Expected misses>=1, got %v", data["misses"])
	}
}

func TestHTTPSetWithTTL(t *testing.T) {
	srv, baseURL := startTestHTTPServer(t, 10)
	defer srv.Stop()

	body := bytes.NewBufferString(`{"value": "temp", "ttl": 1}`)
	resp, err := http.Post(baseURL+"/keys/ttlkey", "application/json", body)
	if err != nil {
		t.Fatalf("POST /keys/ttlkey failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("POST with TTL: expected 201, got %d", resp.StatusCode)
	}

	// Should exist immediately
	resp, err = http.Get(baseURL + "/keys/ttlkey")
	if err != nil {
		t.Fatalf("GET /keys/ttlkey failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET after TTL set: expected 200, got %d", resp.StatusCode)
	}

	// Wait for expiration
	time.Sleep(1100 * time.Millisecond)

	resp, err = http.Get(baseURL + "/keys/ttlkey")
	if err != nil {
		t.Fatalf("GET /keys/ttlkey after expiry failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET after TTL expiry: expected 404, got %d", resp.StatusCode)
	}
}

func TestHTTPSaveAndLoad(t *testing.T) {
	srv, baseURL := startTestHTTPServer(t, 10)
	defer srv.Stop()

	// Set a key
	body := bytes.NewBufferString(`{"value": "persist_me"}`)
	http.Post(baseURL+"/keys/pkey", "application/json", body)

	// SAVE
	resp, err := http.Post(baseURL+"/save", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /save failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST /save: expected 200, got %d", resp.StatusCode)
	}

	// LOAD
	resp, err = http.Post(baseURL+"/load", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /load failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST /load: expected 200, got %d", resp.StatusCode)
	}

	// Verify key still exists
	resp, err = http.Get(baseURL + "/keys/pkey")
	if err != nil {
		t.Fatalf("GET /keys/pkey after load failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET after load: expected 200, got %d", resp.StatusCode)
	}

	// Cleanup snapshot file created by test
	os.Remove("memstash_data.json")
}

func TestHTTPInvalidBody(t *testing.T) {
	srv, baseURL := startTestHTTPServer(t, 10)
	defer srv.Stop()

	body := bytes.NewBufferString(`not json at all`)
	resp, err := http.Post(baseURL+"/keys/bad", "application/json", body)
	if err != nil {
		t.Fatalf("POST /keys/bad failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Invalid body: expected 400, got %d", resp.StatusCode)
	}
}

func TestHTTPConcurrentClients(t *testing.T) {
	srv, baseURL := startTestHTTPServer(t, 100)
	defer srv.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			key := fmt.Sprintf("ckey%d", id)
			value := fmt.Sprintf("cval%d", id)

			// SET
			body := bytes.NewBufferString(fmt.Sprintf(`{"value": "%s"}`, value))
			resp, err := http.Post(baseURL+"/keys/"+key, "application/json", body)
			if err != nil {
				t.Errorf("Client %d POST failed: %v", id, err)
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				t.Errorf("Client %d POST: expected 201, got %d", id, resp.StatusCode)
			}

			// GET
			resp, err = http.Get(baseURL + "/keys/" + key)
			if err != nil {
				t.Errorf("Client %d GET failed: %v", id, err)
				return
			}
			data := decodeJSON(t, resp.Body)
			resp.Body.Close()
			if data["value"] != value {
				t.Errorf("Client %d GET: expected %s, got %v", id, value, data["value"])
			}
		}(i)
	}
	wg.Wait()
}
