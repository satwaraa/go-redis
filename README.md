<p align="center">
  <h1 align="center">⚡ memstash</h1>
  <p align="center">
    A Redis-inspired, in-memory key-value store built from scratch in Go.
    <br />
    LRU eviction · TTL expiration · RESP protocol · REST API · Snapshot persistence
  </p>
</p>

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Configuration](#configuration)
  - [Running](#running)
- [Usage](#usage)
  - [Interactive CLI](#interactive-cli)
  - [TCP Server (RESP Protocol)](#tcp-server-resp-protocol)
  - [HTTP REST API](#http-rest-api)
- [Commands Reference](#commands-reference)
- [REST API Reference](#rest-api-reference)
- [Project Structure](#project-structure)
- [Internals](#internals)
  - [LRU Cache](#lru-cache)
  - [TTL & Expiration](#ttl--expiration)
  - [Persistence](#persistence)
  - [RESP Protocol](#resp-protocol)
- [Docker](#docker)
- [Testing](#testing)
- [Environment Variables](#environment-variables)

---

## Overview

memstash is a lightweight, Redis-compatible in-memory key-value store written entirely in Go with **zero external dependencies** (aside from `godotenv` for configuration). It implements core Redis concepts including LRU eviction, key expiration (TTL), snapshot persistence, and the RESP wire protocol — making it compatible with standard Redis clients over TCP.

It also exposes a **JSON REST API** for web-based integrations and ships with an **interactive CLI** for local development and debugging.

## Features

| Feature | Description |
|---------|-------------|
| **In-Memory Store** | Hash map backed key-value storage with O(1) reads and writes |
| **LRU Eviction** | Doubly-linked list tracks access order; evicts least-recently-used keys when capacity is reached |
| **TTL Expiration** | Per-key time-to-live with lazy deletion on access + background cleaner goroutine |
| **RESP Protocol** | TCP server speaks the Redis Serialization Protocol — works with `redis-cli` and any Redis client |
| **REST API** | JSON-based HTTP API for all store operations |
| **Interactive CLI** | REPL-style command line interface with full command support |
| **Snapshot Persistence** | JSON-based save/load with automatic backup, auto-save, and graceful shutdown saving |
| **Concurrency Safe** | All operations are protected by `sync.RWMutex` for safe concurrent access |
| **Docker Support** | Multi-stage Docker build for minimal production images |

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                      memstash                             │
│                                                          │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐  │
│  │ Interactive  │  │  TCP Server  │  │  HTTP Server   │  │
│  │    CLI       │  │  (RESP)      │  │  (REST/JSON)   │  │
│  │  :stdin      │  │  :6379       │  │  :8080         │  │
│  └──────┬───────┘  └──────┬───────┘  └───────┬────────┘  │
│         │                 │                   │           │
│         └────────────┬────┴───────────────────┘           │
│                      ▼                                    │
│         ┌────────────────────────┐                        │
│         │      Store (Core)      │                        │
│         │  ┌──────────────────┐  │                        │
│         │  │  HashMap + LRU   │  │                        │
│         │  │  Doubly-Linked   │  │                        │
│         │  │     List         │  │                        │
│         │  └──────────────────┘  │                        │
│         │  ┌──────┐ ┌─────────┐  │                        │
│         │  │ TTL  │ │ Persist │  │                        │
│         │  └──────┘ └─────────┘  │                        │
│         └────────────────────────┘                        │
└──────────────────────────────────────────────────────────┘
```

All three interfaces (CLI, TCP, HTTP) share a **single `Store` instance**, meaning changes made through any interface are immediately visible to the others.

## Getting Started

### Prerequisites

- **Go 1.22+** (the project uses Go 1.25, but the HTTP mux pattern matching requires at minimum Go 1.22)

### Installation

```bash
git clone https://github.com/satwaraa/memstash.git
cd memstash
go mod download
```

### Configuration

Create a `.env` file in the project root (or use environment variables):

```env
CAPACITY=10         # Maximum number of keys in the store
TCP_PORT=6379       # Port for the RESP TCP server
HTTP_PORT=8080      # Port for the HTTP REST API (default: 8080)
```

### Running

```bash
go run ./cmd/kvstore/main.go
```

This starts all three interfaces simultaneously:
- **CLI** — interactive prompt in the terminal
- **TCP server** — listening on `TCP_PORT` (default `:6379`)
- **HTTP server** — listening on `HTTP_PORT` (default `:8080`)

---

## Usage

### Interactive CLI

When you run memstash, you're dropped into an interactive REPL:

```
memstash v1.0 - Interactive CLI
Type 'HELP' for commands, 'QUIT' to exit

memstash> SET name "John Doe"
OK
memstash> GET name
"John Doe"
memstash> SETEX session 60 abc123
OK (expires in 60s)
memstash> TTL session
58 (seconds)
memstash> KEYS
[name session]
memstash> STATS
Keys: 2
Capacity: 10
Hits: 2
Misses: 0
Evictions: 0
memstash> QUIT
Goodbye!
```

### TCP Server (RESP Protocol)

The TCP server speaks the [RESP protocol](https://redis.io/docs/reference/protocol-spec/), so you can use any Redis client or `redis-cli`:

```bash
# Using redis-cli
redis-cli -p 6379
127.0.0.1:6379> PING
PONG
127.0.0.1:6379> SET mykey myvalue
OK
127.0.0.1:6379> GET mykey
"myvalue"

# Using netcat
echo "PING" | nc localhost 6379
# +PONG

echo "SET hello world" | nc localhost 6379
# +OK
```

### HTTP REST API

The REST API uses JSON for all requests and responses:

```bash
# Set a key
curl -X POST http://localhost:8080/keys/name \
  -H "Content-Type: application/json" \
  -d '{"value": "memstash"}'
# {"key":"name","status":"OK"}

# Set a key with TTL (expires in 60 seconds)
curl -X POST http://localhost:8080/keys/session \
  -H "Content-Type: application/json" \
  -d '{"value": "token123", "ttl": 60}'
# {"key":"session","status":"OK"}

# Get a key
curl http://localhost:8080/keys/name
# {"key":"name","value":"memstash"}

# Delete a key
curl -X DELETE http://localhost:8080/keys/name
# {"key":"name","status":"OK"}

# List all keys
curl http://localhost:8080/keys
# {"count":1,"keys":["session"]}

# Get store statistics
curl http://localhost:8080/stats
# {"capacity":10,"evictions":0,"hits":1,"keys":1,"misses":0}

# Save snapshot to disk
curl -X POST http://localhost:8080/save
# {"status":"OK"}

# Load snapshot from disk
curl -X POST http://localhost:8080/load
# {"status":"OK"}
```

---

## Commands Reference

Full list of commands supported across CLI and TCP:

| Command | Syntax | Description |
|---------|--------|-------------|
| `SET` | `SET <key> <value>` | Set a key-value pair. Overwrites if key exists. |
| `GET` | `GET <key>` | Retrieve the value for a key. |
| `DEL` / `DELETE` | `DEL <key>` | Delete a key from the store. |
| `EXISTS` | `EXISTS <key>` | Check if a key exists. Returns `1` or `0`. |
| `KEYS` | `KEYS` | List all keys in the store. |
| `CLEAR` | `CLEAR` | Remove all keys from the store. |
| `SETEX` | `SETEX <key> <seconds> <value>` | Set a key with an expiration time in seconds. |
| `TTL` | `TTL <key>` | Get remaining time-to-live in seconds. `-1` = no expiry, `-2` = key not found. |
| `EXPIRE` | `EXPIRE <key> <seconds>` | Set an expiration on an existing key. |
| `SAVE` | `SAVE` | Persist the current store to a JSON snapshot file. |
| `LOAD` | `LOAD` | Load the store from a snapshot file. |
| `STATS` | `STATS` | Display store statistics (keys, capacity, hits, misses, evictions). |
| `PING` | `PING` | Test connection (TCP only). Returns `PONG`. |
| `HELP` | `HELP` | Display the help message. |
| `QUIT` | `QUIT` | Close the connection (TCP) or exit the CLI. |

---

## REST API Reference

| Method | Endpoint | Body | Response | Status Codes |
|--------|----------|------|----------|--------------|
| `POST` | `/keys/{key}` | `{"value": "...", "ttl": N}` | `{"status": "OK", "key": "..."}` | `201` Created, `400` Bad Request |
| `GET` | `/keys/{key}` | — | `{"key": "...", "value": "..."}` | `200` OK, `404` Not Found |
| `DELETE` | `/keys/{key}` | — | `{"status": "OK", "key": "..."}` | `200` OK, `404` Not Found |
| `GET` | `/keys` | — | `{"keys": [...], "count": N}` | `200` OK |
| `GET` | `/stats` | — | `{"keys": N, "capacity": N, ...}` | `200` OK |
| `POST` | `/save` | — | `{"status": "OK"}` | `200` OK, `500` Error |
| `POST` | `/load` | — | `{"status": "OK"}` | `200` OK, `500` Error |

> **Note:** The `ttl` field in the `POST /keys/{key}` body is optional. When provided, the key will automatically expire after the specified number of seconds.

---

## Project Structure

```
memstash/
├── cmd/
│   └── kvstore/
│       └── main.go              # Entry point — starts CLI, TCP, and HTTP servers
├── docker/
│   ├── dockerfile               # Multi-stage Docker build
│   └── docker-compose.yaml      # Docker Compose configuration
├── env/
│   └── env.go                   # Environment variable loading
├── internal/
│   ├── cli/
│   │   └── cli.go               # Interactive CLI (REPL)
│   ├── protocol/
│   │   └── resp.go              # RESP protocol formatters
│   ├── server/
│   │   ├── server.go            # TCP server (RESP wire protocol)
│   │   └── http_server.go       # HTTP REST API server
│   └── store/
│       ├── store.go             # Core key-value store with LRU eviction
│       ├── lru.go               # Doubly-linked list for LRU tracking
│       ├── ttl.go               # TTL expiration logic + background cleaner
│       └── persistence.go       # JSON snapshot save/load + auto-save
├── tests/
│   ├── store_test.go            # Store unit tests (55+ test cases)
│   ├── server_test.go           # TCP server integration tests
│   └── http_server_test.go      # HTTP server integration tests
├── .env                         # Environment configuration
├── .gitignore
├── go.mod
└── go.sum
```

---

## Internals

### LRU Cache

The store uses a **hash map + doubly-linked list** combination for O(1) operations:

- **Hash Map** (`map[string]*Node`) — provides O(1) key lookup
- **Doubly-Linked List** (`LruList`) — maintains access order, with the most recently used key at the head

When the store reaches capacity, the **tail node** (least recently used) is evicted to make room for new entries. Every `GET` or `SET` on an existing key moves it to the head of the list.

```
HEAD (most recent) ←→ Node ←→ Node ←→ ... ←→ TAIL (least recent)
                                                    ↑ evicted first
```

### TTL & Expiration

Keys can have an optional expiration time:

- **Lazy deletion** — expired keys are removed on access (`GET` checks expiration and returns `ErrKeyExpired`)
- **Background cleaner** — a goroutine periodically walks the LRU list and removes expired keys (configurable interval, default: 1 minute)

```go
// Set a key that expires in 60 seconds
store.SetWithTTL("session", "token123", 60 * time.Second)

// Set expiration on an existing key
store.SetExpiry("mykey", 30 * time.Second)

// Check remaining TTL
ttl, err := store.GetTTL("session") // returns time.Duration
```

### Persistence

memstash persists data to JSON snapshot files:

- **Manual save/load** — `SAVE` and `LOAD` commands
- **Auto-save** — background goroutine saves at configurable intervals (default: 1 minute)
- **Graceful shutdown** — captures `SIGINT`/`SIGTERM` and saves before exiting
- **Backup safety** — reads existing file before overwriting; wipes old data before writing new snapshot

Snapshot format:
```json
{
  "version": "1.0",
  "capacity": 10,
  "entries": [
    {
      "key": "name",
      "value": "memstash",
      "expire_at": "2026-02-25T00:00:00Z"
    }
  ]
}
```

Entries are stored in LRU order (head → tail), so loading a snapshot preserves the original access ordering. Expired entries are skipped during both save and load.

### RESP Protocol

The TCP server implements a subset of the [Redis Serialization Protocol (RESP)](https://redis.io/docs/reference/protocol-spec/):

| Type | Prefix | Example |
|------|--------|---------|
| Simple String | `+` | `+OK\r\n` |
| Error | `-` | `-ERR unknown command\r\n` |
| Integer | `:` | `:1\r\n` |
| Bulk String | `$` | `$3\r\nbar\r\n` |
| Null | `$-1` | `$-1\r\n` |
| Array | `*` | `*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n` |

---

## Docker

### Build the image

```bash
docker build -t memstash:latest -f docker/dockerfile .
```

### Run with Docker Compose

```bash
docker compose -f docker/docker-compose.yaml up
```

### Run directly

```bash
docker run -e CAPACITY=100 -e TCP_PORT=6379 -e HTTP_PORT=8080 \
  -p 6379:6379 -p 8080:8080 \
  memstash:latest
```

> **Note:** The Docker image uses a multi-stage build (Go builder → Alpine) for a minimal final image.

---

## Testing

Run the full test suite:

```bash
go test ./tests/ -v -count=1
```

The test suite includes **55+ tests** across three categories:

| Category | File | Tests |
|----------|------|-------|
| Store unit tests | `tests/store_test.go` | LRU eviction, TTL, persistence, concurrency, edge cases |
| TCP server tests | `tests/server_test.go` | RESP protocol, all commands, concurrent clients, shared store |
| HTTP server tests | `tests/http_server_test.go` | All REST endpoints, TTL expiry, error handling, concurrent access |

```bash
# Run only store tests
go test ./tests/ -run TestStore -v

# Run only HTTP tests
go test ./tests/ -run TestHTTP -v

# Run only TCP server tests
go test ./tests/ -run TestServer -v
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CAPACITY` | Yes* | — | Maximum number of keys the store can hold |
| `Memory` | Yes* | — | Alternative to CAPACITY (one of the two is required) |
| `TCP_PORT` | Yes | — | Port for the RESP TCP server |
| `HTTP_PORT` | No | `8080` | Port for the HTTP REST API |

> *Either `CAPACITY` or `Memory` must be provided.

---

<p align="center">
  Built with ❤️ in Go
</p>
