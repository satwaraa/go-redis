package server

import (
	"encoding/json"
	"fmt"
	"log"
	"memstash/internal/store"
	"net"
	"net/http"
	"strings"
	"time"
)

// HTTPServer exposes the store via a JSON REST API.
type HTTPServer struct {
	store        *store.Store
	port         int
	snapshotPath string
	server       *http.Server
	listener     net.Listener
}

// NewHTTPServer creates a new HTTP server sharing the given store.
func NewHTTPServer(s *store.Store, port int) *HTTPServer {
	return &HTTPServer{
		store:        s,
		port:         port,
		snapshotPath: "memstash_data.json",
	}
}

// Start binds to the configured port and serves HTTP requests.
func (h *HTTPServer) Start() {
	mux := h.routes()
	h.server = &http.Server{Handler: mux}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", h.port))
	if err != nil {
		log.Fatalf("HTTP server failed to start: %v", err)
	}
	h.listener = ln
	log.Printf("HTTP server listening on %s", ln.Addr().String())

	if err := h.server.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP server error: %v", err)
	}
}

// Stop gracefully shuts down the HTTP server.
func (h *HTTPServer) Stop() {
	if h.server != nil {
		h.server.Close()
	}
}

// Addr returns the listener's address (useful for tests with port 0).
func (h *HTTPServer) Addr() net.Addr {
	if h.listener != nil {
		return h.listener.Addr()
	}
	return nil
}

// StartAndReady starts the server and signals via the returned channel
// once the listener is bound (useful for tests).
func (h *HTTPServer) StartAndReady() <-chan struct{} {
	ready := make(chan struct{})
	go func() {
		mux := h.routes()
		h.server = &http.Server{Handler: mux}

		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", h.port))
		if err != nil {
			log.Fatalf("HTTP server failed to start: %v", err)
		}
		h.listener = ln
		log.Printf("HTTP server listening on %s", ln.Addr().String())
		close(ready)

		if err := h.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	return ready
}

// routes configures the HTTP mux with all endpoints.
func (h *HTTPServer) routes() *http.ServeMux {
	mux := http.NewServeMux()

	// Key-specific operations: /keys/{key}
	mux.HandleFunc("POST /keys/{key}", h.handleSetKey)
	mux.HandleFunc("GET /keys/{key}", h.handleGetKey)
	mux.HandleFunc("DELETE /keys/{key}", h.handleDeleteKey)

	// List all keys
	mux.HandleFunc("GET /keys", h.handleListKeys)

	// Stats
	mux.HandleFunc("GET /stats", h.handleGetStats)

	// Persistence
	mux.HandleFunc("POST /save", h.handleSave)
	mux.HandleFunc("POST /load", h.handleLoad)

	return mux
}

// ── JSON helpers ────────────────────────────────────────────────────────

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	jsonResponse(w, status, map[string]string{"error": msg})
}

// ── Handlers ────────────────────────────────────────────────────────────

// POST /keys/{key}
// Body: {"value": "...", "ttl": <optional seconds>}
func (h *HTTPServer) handleSetKey(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		jsonError(w, http.StatusBadRequest, "key is required")
		return
	}

	var body struct {
		Value string `json:"value"`
		TTL   *int   `json:"ttl,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if body.TTL != nil && *body.TTL > 0 {
		ttl := time.Duration(*body.TTL) * time.Second
		if err := h.store.SetWithTTL(key, body.Value, ttl); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		if err := h.store.Set(key, body.Value); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	jsonResponse(w, http.StatusCreated, map[string]string{
		"status": "OK",
		"key":    key,
	})
}

// GET /keys/{key}
func (h *HTTPServer) handleGetKey(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		jsonError(w, http.StatusBadRequest, "key is required")
		return
	}

	value, err := h.store.Get(key)
	if err != nil {
		// Distinguish between not-found and expired
		msg := err.Error()
		if strings.Contains(msg, "not found") || strings.Contains(msg, "expired") {
			jsonError(w, http.StatusNotFound, fmt.Sprintf("key '%s' not found", key))
			return
		}
		jsonError(w, http.StatusInternalServerError, msg)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"key":   key,
		"value": value,
	})
}

// DELETE /keys/{key}
func (h *HTTPServer) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		jsonError(w, http.StatusBadRequest, "key is required")
		return
	}

	err := h.store.Delete(key)
	if err != nil {
		jsonError(w, http.StatusNotFound, fmt.Sprintf("key '%s' not found", key))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"status": "OK",
		"key":    key,
	})
}

// GET /keys
func (h *HTTPServer) handleListKeys(w http.ResponseWriter, r *http.Request) {
	keys := h.store.Keys()
	if keys == nil {
		keys = []string{}
	}
	jsonResponse(w, http.StatusOK, map[string]any{
		"keys":  keys,
		"count": len(keys),
	})
}

// GET /stats
func (h *HTTPServer) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.store.Stats()
	jsonResponse(w, http.StatusOK, map[string]any{
		"keys":      stats.Keys,
		"capacity":  stats.Capacity,
		"hits":      stats.Hits,
		"misses":    stats.Misses,
		"evictions": stats.Evictions,
	})
}

// POST /save
func (h *HTTPServer) handleSave(w http.ResponseWriter, r *http.Request) {
	if err := h.store.SaveSnapshot(h.snapshotPath); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "OK"})
}

// POST /load
func (h *HTTPServer) handleLoad(w http.ResponseWriter, r *http.Request) {
	if err := h.store.LoadSnapshot(h.snapshotPath); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "OK"})
}
