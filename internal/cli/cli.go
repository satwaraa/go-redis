package cli

import (
	"bufio"
	"fmt"
	"goredis/internal/store"
	"os"
	"strconv"
	"strings"
	"time"
)

type CLI struct {
	store  *store.Store
	reader *bufio.Reader
}

func NewCLI(s *store.Store) *CLI {
	return &CLI{
		store:  s,
		reader: bufio.NewReader(os.Stdin),
	}
}

func (c *CLI) parseCommand(input string) (string, []string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil
	}
	cmd := strings.ToUpper(parts[0])
	args := parts[1:]
	return cmd, args
}

func (c *CLI) Start() {
	fmt.Println("GoRedis v1.0 - Interactive CLI")
	fmt.Println("Type 'HELP' for commands, 'QUIT' to exit")
	fmt.Println()

	for {
		fmt.Print("goredis> ")

		input, err := c.reader.ReadString('\n')
		if err != nil {
			fmt.Println("Goodbye! (stdin closed)")
			return
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		cmd, args := c.parseCommand(input)

		if cmd == "QUIT" || cmd == "EXIT" {
			fmt.Println("Goodbye!")
			break
		}

		c.executeCommand(cmd, args)
	}
}
func (c *CLI) executeCommand(cmd string, args []string) {
	switch cmd {
	case "SET":
		c.handleSet(args)
	case "GET":
		c.handleGet(args)
	case "DELETE", "DEL":
		c.handleDelete(args)
	case "SETEX":
		c.handleSetEx(args)
	case "TTL":
		c.handleTTL(args)
	case "HELP":
		c.handleHelp(args)
	case "STATS":
		c.handleStats(args)
	default:
		fmt.Printf("Unknown command: %s. Type HELP for commands.\n", cmd)
	}
}

func (c *CLI) handleStats(args []string) {
	stats := c.store.Stats()
	fmt.Println("Keys:", stats.Keys)
	fmt.Println("Capacity:", stats.Capacity)
	fmt.Println("Hits:", stats.Hits)
	fmt.Println("Misses:", stats.Misses)
	fmt.Println("Evictions:", stats.Evictions)
}

func (c *CLI) handleSet(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: SET <key> <value>")
		return
	}

	key := args[0]
	value := strings.Join(args[1:], " ")

	err := c.store.Set(key, value)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("OK")
	}
}

func (c *CLI) handleGet(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: GET <key>")
		return
	}

	key := args[0]
	value, err := c.store.Get(key)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("\"%s\"\n", value)
	}
}

func (c *CLI) handleDelete(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: DELETE <key>")
		return
	}

	key := args[0]
	err := c.store.Delete(key)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("OK")
	}
}

func (c *CLI) handleSetEx(args []string) {
	if len(args) < 3 {
		fmt.Println("Usage: SETEX <key> <seconds> <value>")
		return
	}

	key := args[0]
	seconds, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Println("Invalid seconds value")
		return
	}

	value := strings.Join(args[2:], " ")
	ttl := time.Duration(seconds) * time.Second

	err = c.store.SetWithTTL(key, value, ttl)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("OK (expires in %ds)\n", seconds)
	}
}

func (c *CLI) handleTTL(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: TTL <key>")
		return
	}

	key := args[0]
	ttl, err := c.store.GetTTL(key)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		if ttl < 0 {
			fmt.Println("-1 (no expiration)")
		} else {
			fmt.Printf("%d (seconds)\n", int(ttl.Seconds()))
		}
	}
}

func (c *CLI) handleHelp(args []string) {
	fmt.Println(`
Available Commands:
  SET <key> <value>          - Set a key-value pair
  GET <key>                  - Get value by key
  DELETE <key>               - Delete a key
  EXISTS <key>               - Check if key exists
  KEYS                       - List all keys
  CLEAR                      - Remove all keys

  SETEX <key> <sec> <value>  - Set with expiration (seconds)
  TTL <key>                  - Get time to live in seconds
  EXPIRE <key> <seconds>     - Set expiration on existing key

  SAVE [file]                - Save snapshot to disk
  LOAD [file]                - Load snapshot from disk

  STATS                      - Show statistics
  HELP                       - Show this help
  QUIT                       - Exit CLI
`)
}
