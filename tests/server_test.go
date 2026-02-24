package tests

import (
	"bufio"
	"fmt"
	"goredis/internal/server"
	"goredis/internal/store"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// helper: start a server on port 0 (OS picks a free port), return server + address
func startTestServer(t *testing.T, capacity int) (*server.Server, string) {
	t.Helper()
	s := store.NewStore(capacity)
	srv := server.NewServer(s, 0)
	ready := srv.StartAndReady()
	<-ready
	addr := srv.Addr().String()
	return srv, addr
}

// helper: dial and return a reader + conn
func dialServer(t *testing.T, addr string) (net.Conn, *bufio.Reader) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	reader := bufio.NewReader(conn)
	return conn, reader
}

// helper: send a command and read the full RESP response
func sendCommand(conn net.Conn, reader *bufio.Reader, cmd string) string {
	fmt.Fprintf(conn, "%s\r\n", cmd)
	return readResponse(reader)
}

// readResponse reads one complete RESP response
func readResponse(reader *bufio.Reader) string {
	line, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	// Simple string (+), Error (-), Integer (:)
	if len(line) > 0 && (line[0] == '+' || line[0] == '-' || line[0] == ':') {
		return line
	}

	// Bulk string ($)
	if len(line) > 0 && line[0] == '$' {
		sizeStr := strings.TrimSpace(line[1:])
		if sizeStr == "-1" {
			return line // null bulk string
		}
		// Read the data line
		data, _ := reader.ReadString('\n')
		return line + data
	}

	// Array (*)
	if len(line) > 0 && line[0] == '*' {
		countStr := strings.TrimSpace(line[1:])
		var count int
		fmt.Sscanf(countStr, "%d", &count)
		result := line
		for i := 0; i < count; i++ {
			result += readResponse(reader)
		}
		return result
	}

	return line
}

// ==================== TESTS ====================

func TestServerPing(t *testing.T) {
	srv, addr := startTestServer(t, 10)
	defer srv.Stop()

	conn, reader := dialServer(t, addr)
	defer conn.Close()

	resp := sendCommand(conn, reader, "PING")
	if resp != "+PONG\r\n" {
		t.Errorf("Expected +PONG\\r\\n, got %q", resp)
	}
}

func TestServerSetAndGet(t *testing.T) {
	srv, addr := startTestServer(t, 10)
	defer srv.Stop()

	conn, reader := dialServer(t, addr)
	defer conn.Close()

	// SET
	resp := sendCommand(conn, reader, "SET foo bar")
	if resp != "+OK\r\n" {
		t.Errorf("SET: expected +OK\\r\\n, got %q", resp)
	}

	// GET
	resp = sendCommand(conn, reader, "GET foo")
	if resp != "$3\r\nbar\r\n" {
		t.Errorf("GET: expected $3\\r\\nbar\\r\\n, got %q", resp)
	}
}

func TestServerGetNonExistent(t *testing.T) {
	srv, addr := startTestServer(t, 10)
	defer srv.Stop()

	conn, reader := dialServer(t, addr)
	defer conn.Close()

	resp := sendCommand(conn, reader, "GET missing")
	if resp != "$-1\r\n" {
		t.Errorf("Expected $-1\\r\\n, got %q", resp)
	}
}

func TestServerDelete(t *testing.T) {
	srv, addr := startTestServer(t, 10)
	defer srv.Stop()

	conn, reader := dialServer(t, addr)
	defer conn.Close()

	sendCommand(conn, reader, "SET mykey myval")

	resp := sendCommand(conn, reader, "DEL mykey")
	if resp != ":1\r\n" {
		t.Errorf("DEL existing: expected :1\\r\\n, got %q", resp)
	}

	resp = sendCommand(conn, reader, "DEL mykey")
	if resp != ":0\r\n" {
		t.Errorf("DEL non-existent: expected :0\\r\\n, got %q", resp)
	}

	resp = sendCommand(conn, reader, "GET mykey")
	if resp != "$-1\r\n" {
		t.Errorf("GET after DEL: expected $-1\\r\\n, got %q", resp)
	}
}

func TestServerExists(t *testing.T) {
	srv, addr := startTestServer(t, 10)
	defer srv.Stop()

	conn, reader := dialServer(t, addr)
	defer conn.Close()

	sendCommand(conn, reader, "SET hello world")

	resp := sendCommand(conn, reader, "EXISTS hello")
	if resp != ":1\r\n" {
		t.Errorf("EXISTS existing: expected :1\\r\\n, got %q", resp)
	}

	resp = sendCommand(conn, reader, "EXISTS nope")
	if resp != ":0\r\n" {
		t.Errorf("EXISTS non-existent: expected :0\\r\\n, got %q", resp)
	}
}

func TestServerSetex(t *testing.T) {
	srv, addr := startTestServer(t, 10)
	defer srv.Stop()

	conn, reader := dialServer(t, addr)
	defer conn.Close()

	resp := sendCommand(conn, reader, "SETEX temp 1 tempval")
	if resp != "+OK\r\n" {
		t.Errorf("SETEX: expected +OK\\r\\n, got %q", resp)
	}

	// Should exist immediately
	resp = sendCommand(conn, reader, "GET temp")
	if resp != "$7\r\ntempval\r\n" {
		t.Errorf("GET after SETEX: expected $7\\r\\ntempval\\r\\n, got %q", resp)
	}

	// Wait for expiration
	time.Sleep(1100 * time.Millisecond)

	resp = sendCommand(conn, reader, "GET temp")
	if resp != "$-1\r\n" {
		t.Errorf("GET after expiry: expected $-1\\r\\n, got %q", resp)
	}
}

func TestServerTTL(t *testing.T) {
	srv, addr := startTestServer(t, 10)
	defer srv.Stop()

	conn, reader := dialServer(t, addr)
	defer conn.Close()

	sendCommand(conn, reader, "SETEX key 10 val")

	resp := sendCommand(conn, reader, "TTL key")
	// Should be around 9-10 seconds
	if !strings.HasPrefix(resp, ":") {
		t.Errorf("TTL: expected integer response, got %q", resp)
	}

	// Non-existent key
	resp = sendCommand(conn, reader, "TTL nope")
	if resp != ":-2\r\n" {
		t.Errorf("TTL non-existent: expected :-2\\r\\n, got %q", resp)
	}
}

func TestServerKeys(t *testing.T) {
	srv, addr := startTestServer(t, 10)
	defer srv.Stop()

	conn, reader := dialServer(t, addr)
	defer conn.Close()

	sendCommand(conn, reader, "SET alpha 1")
	sendCommand(conn, reader, "SET beta 2")
	sendCommand(conn, reader, "SET gamma 3")

	resp := sendCommand(conn, reader, "KEYS")
	// Should be a RESP array with 3 elements
	if !strings.HasPrefix(resp, "*3\r\n") {
		t.Errorf("KEYS: expected *3\\r\\n prefix, got %q", resp)
	}
	if !strings.Contains(resp, "alpha") || !strings.Contains(resp, "beta") || !strings.Contains(resp, "gamma") {
		t.Errorf("KEYS: expected all keys in response, got %q", resp)
	}
}

func TestServerKeysEmpty(t *testing.T) {
	srv, addr := startTestServer(t, 10)
	defer srv.Stop()

	conn, reader := dialServer(t, addr)
	defer conn.Close()

	resp := sendCommand(conn, reader, "KEYS")
	if resp != "$-1\r\n" {
		t.Errorf("KEYS empty: expected $-1\\r\\n, got %q", resp)
	}
}

func TestServerQuit(t *testing.T) {
	srv, addr := startTestServer(t, 10)
	defer srv.Stop()

	conn, reader := dialServer(t, addr)
	defer conn.Close()

	resp := sendCommand(conn, reader, "QUIT")
	if resp != "+OK\r\n" {
		t.Errorf("QUIT: expected +OK\\r\\n, got %q", resp)
	}

	// Connection should be closed by server; further reads should fail
	_, err := reader.ReadString('\n')
	if err == nil {
		t.Error("Expected connection to be closed after QUIT")
	}
}

func TestServerUnknownCommand(t *testing.T) {
	srv, addr := startTestServer(t, 10)
	defer srv.Stop()

	conn, reader := dialServer(t, addr)
	defer conn.Close()

	resp := sendCommand(conn, reader, "FOOBAR")
	if !strings.HasPrefix(resp, "-ERR") {
		t.Errorf("Unknown command: expected error response, got %q", resp)
	}
}

func TestServerConcurrentClients(t *testing.T) {
	srv, addr := startTestServer(t, 100)
	defer srv.Stop()

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, reader := dialServer(t, addr)
			defer conn.Close()

			key := fmt.Sprintf("key%d", id)
			value := fmt.Sprintf("value%d", id)

			// SET
			resp := sendCommand(conn, reader, fmt.Sprintf("SET %s %s", key, value))
			if resp != "+OK\r\n" {
				t.Errorf("Client %d SET: expected +OK, got %q", id, resp)
			}

			// GET
			resp = sendCommand(conn, reader, fmt.Sprintf("GET %s", key))
			expected := fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
			if resp != expected {
				t.Errorf("Client %d GET: expected %q, got %q", id, expected, resp)
			}
		}(i)
	}

	wg.Wait()
}

func TestServerSharedStore(t *testing.T) {
	// Verify two TCP clients share the same store
	srv, addr := startTestServer(t, 10)
	defer srv.Stop()

	// Client 1: set a key
	conn1, reader1 := dialServer(t, addr)
	defer conn1.Close()
	sendCommand(conn1, reader1, "SET shared_key shared_val")

	// Client 2: get the same key
	conn2, reader2 := dialServer(t, addr)
	defer conn2.Close()
	resp := sendCommand(conn2, reader2, "GET shared_key")

	if resp != "$10\r\nshared_val\r\n" {
		t.Errorf("Shared store: expected $10\\r\\nshared_val\\r\\n, got %q", resp)
	}
}
