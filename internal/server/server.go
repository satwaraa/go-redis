package server

import (
	"bufio"
	"fmt"
	"log"
	"memstash/internal/protocol"
	"memstash/internal/store"
	"net"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	store        *store.Store
	listener     net.Listener
	port         int
	snapshotPath string
}

func NewServer(s *store.Store, port int) *Server {
	return &Server{
		store:        s,
		port:         port,
		snapshotPath: "memstash_data.json",
	}
}

// Start binds to the configured TCP port and accepts connections.
func (srv *Server) Start() {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", srv.port))
	if err != nil {
		log.Fatalf("TCP server failed to start: %v", err)
	}
	srv.listener = ln
	log.Printf("TCP server listening on :%d", srv.port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			// listener was closed (graceful shutdown)
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			log.Printf("Accept error: %v", err)
			continue
		}
		go srv.handleConnection(conn)
	}
}

// Stop closes the listener for graceful shutdown.
func (srv *Server) Stop() {
	if srv.listener != nil {
		srv.listener.Close()
	}
}

// Addr returns the listener's address (useful for tests with port 0).
func (srv *Server) Addr() net.Addr {
	if srv.listener != nil {
		return srv.listener.Addr()
	}
	return nil
}

// StartAndReady starts the server and signals via the returned channel
// once the listener is bound (useful for tests).
func (srv *Server) StartAndReady() <-chan struct{} {
	ready := make(chan struct{})
	go func() {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", srv.port))
		if err != nil {
			log.Fatalf("TCP server failed to start: %v", err)
		}
		srv.listener = ln
		log.Printf("TCP server listening on %s", ln.Addr().String())
		close(ready)

		for {
			conn, err := ln.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}
				log.Printf("Accept error: %v", err)
				continue
			}
			go srv.handleConnection(conn)
		}
	}()
	return ready
}

func (srv *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := strings.ToUpper(parts[0])
		args := parts[1:]

		response := srv.executeCommand(cmd, args)
		conn.Write([]byte(response))

		if cmd == "QUIT" {
			return
		}
	}
}

func (srv *Server) executeCommand(cmd string, args []string) string {
	switch cmd {
	case "PING":
		return protocol.FormatPong()

	case "SET":
		return srv.handleSet(args)

	case "GET":
		return srv.handleGet(args)

	case "DEL", "DELETE":
		return srv.handleDelete(args)

	case "EXISTS":
		return srv.handleExists(args)

	case "SETEX":
		return srv.handleSetEx(args)

	case "TTL":
		return srv.handleTTL(args)

	case "KEYS":
		return srv.handleKeys()

	case "SAVE":
		return srv.handleSave()

	case "LOAD":
		return srv.handleLoad()

	case "CLEAR":
		srv.store.Clear()
		return protocol.FormatOK()

	case "EXPIRE":
		return srv.handleExpire(args)

	case "STATS":
		return srv.handleStats()

	case "HELP":
		return srv.handleHelp()

	case "QUIT":
		return protocol.FormatOK()

	default:
		return protocol.FormatError(fmt.Sprintf("unknown command '%s'", cmd))
	}
}

func (srv *Server) handleSet(args []string) string {
	if len(args) < 2 {
		return protocol.FormatError("wrong number of arguments for 'SET' command")
	}
	key := args[0]
	value := strings.Join(args[1:], " ")

	err := srv.store.Set(key, value)
	if err != nil {
		return protocol.FormatError(err.Error())
	}
	return protocol.FormatOK()
}

func (srv *Server) handleGet(args []string) string {
	if len(args) < 1 {
		return protocol.FormatError("wrong number of arguments for 'GET' command")
	}
	key := args[0]
	value, err := srv.store.Get(key)
	if err != nil {
		return protocol.FormatNull()
	}
	return protocol.FormatBulkString(value)
}

func (srv *Server) handleDelete(args []string) string {
	if len(args) < 1 {
		return protocol.FormatError("wrong number of arguments for 'DEL' command")
	}
	key := args[0]
	err := srv.store.Delete(key)
	if err != nil {
		return protocol.FormatInteger(0)
	}
	return protocol.FormatInteger(1)
}

func (srv *Server) handleExists(args []string) string {
	if len(args) < 1 {
		return protocol.FormatError("wrong number of arguments for 'EXISTS' command")
	}
	key := args[0]
	if srv.store.Exists(key) {
		return protocol.FormatInteger(1)
	}
	return protocol.FormatInteger(0)
}

func (srv *Server) handleSetEx(args []string) string {
	if len(args) < 3 {
		return protocol.FormatError("wrong number of arguments for 'SETEX' command")
	}
	key := args[0]
	seconds, err := strconv.Atoi(args[1])
	if err != nil {
		return protocol.FormatError("value is not an integer or out of range")
	}
	value := strings.Join(args[2:], " ")
	ttl := time.Duration(seconds) * time.Second

	err = srv.store.SetWithTTL(key, value, ttl)
	if err != nil {
		return protocol.FormatError(err.Error())
	}
	return protocol.FormatOK()
}

func (srv *Server) handleTTL(args []string) string {
	if len(args) < 1 {
		return protocol.FormatError("wrong number of arguments for 'TTL' command")
	}
	key := args[0]
	ttl, err := srv.store.GetTTL(key)
	if err != nil {
		return protocol.FormatInteger(-2) // key does not exist
	}
	return protocol.FormatInteger(int64(ttl.Seconds()))
}

func (srv *Server) handleKeys() string {
	keys := srv.store.Keys()
	if len(keys) == 0 {
		return protocol.FormatNull()
	}
	// Return as multiple bulk strings (simplified array)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("*%d\r\n", len(keys)))
	for _, k := range keys {
		b.WriteString(protocol.FormatBulkString(k))
	}
	return b.String()
}

func (srv *Server) handleSave() string {
	err := srv.store.SaveSnapshot(srv.snapshotPath)
	if err != nil {
		return protocol.FormatError(err.Error())
	}
	return protocol.FormatOK()
}

func (srv *Server) handleLoad() string {
	err := srv.store.LoadSnapshot(srv.snapshotPath)
	if err != nil {
		return protocol.FormatError(err.Error())
	}
	return protocol.FormatOK()
}

func (srv *Server) handleExpire(args []string) string {
	if len(args) < 2 {
		return protocol.FormatError("wrong number of arguments for 'EXPIRE' command")
	}
	key := args[0]
	seconds, err := strconv.Atoi(args[1])
	if err != nil {
		return protocol.FormatError("value is not an integer or out of range")
	}
	ttl := time.Duration(seconds) * time.Second
	err = srv.store.SetExpiry(key, ttl)
	if err != nil {
		return protocol.FormatInteger(0)
	}
	return protocol.FormatInteger(1)
}

func (srv *Server) handleStats() string {
	stats := srv.store.Stats()
	var b strings.Builder
	b.WriteString(fmt.Sprintf("keys:%d\r\n", stats.Keys))
	b.WriteString(fmt.Sprintf("capacity:%d\r\n", stats.Capacity))
	b.WriteString(fmt.Sprintf("hits:%d\r\n", stats.Hits))
	b.WriteString(fmt.Sprintf("misses:%d\r\n", stats.Misses))
	b.WriteString(fmt.Sprintf("evictions:%d\r\n", stats.Evictions))
	return protocol.FormatBulkString(b.String())
}

func (srv *Server) handleHelp() string {
	help := `Commands:
  PING                        - Test connection
  SET <key> <value>           - Set a key-value pair
  GET <key>                   - Get value by key
  DEL <key>                   - Delete a key
  EXISTS <key>                - Check if key exists (1/0)
  SETEX <key> <sec> <value>   - Set with expiration
  TTL <key>                   - Get time to live
  EXPIRE <key> <seconds>      - Set expiration on key
  KEYS                        - List all keys
  SAVE                        - Save snapshot to disk
  LOAD                        - Load snapshot from disk
  CLEAR                       - Remove all keys
  STATS                       - Show statistics
  HELP                        - Show this help
  QUIT                        - Close connection`
	return protocol.FormatBulkString(help)
}
