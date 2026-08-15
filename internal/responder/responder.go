package responder

import (
	"fmt"
)

type Store interface {
	Set(key string, value string)
	Get(key string) (string, bool)
	Del(keys ...string) int
	Exists(keys ...string) int
}

func GetResponse(message [][]byte, store Store) ([]byte, error) {
	// edge case if message is *0\r\n
	if len(message) == 0 {
		return nil, nil
	}

	command := string(message[0])
	switch command {
	case "PING":
		return []byte("+PONG\r\n"), nil
	case "ECHO":
		if len(message) != 2 {
			return []byte("-ERR wrong number of arguments for 'echo' command\r\n"), nil
		}
		return fmt.Appendf(nil, "$%d\r\n%s\r\n", len(message[1]), string(message[1])), nil
	case "SET":
		if len(message) != 3 {
			return []byte("-ERR wrong number of arguments for 'set' command\r\n"), nil
		}
		store.Set(string(message[1]), string(message[2]))
		return []byte("+OK\r\n"), nil
	case "GET":
		if len(message) != 2 {
			return []byte("-ERR wrong number of arguments for 'get' command\r\n"), nil
		}

		value, exists := store.Get(string(message[1]))
		if !exists {
			return []byte("$-1\r\n"), nil
		}
		return fmt.Appendf(nil, "$%d\r\n%s\r\n", len(value), value), nil
	case "DEL":
		if len(message) < 2 {
			return []byte("-ERR wrong number of arguments for 'del' command\r\n"), nil
		}

		// need to get keys, these will be strings
		keys := make([]string, len(message)-1)
		for i := 1; i < len(message); i++ {
			keys[i-1] = string(message[i])
		}

		deleted := store.Del(keys...)
		return fmt.Appendf(nil, ":%d\r\n", deleted), nil
	case "EXISTS":
		if len(message) < 2 {
			return []byte("-ERR wrong number of arguments for 'exists' command\r\n"), nil
		}

		// need to get keys, these will be strings
		keys := make([]string, len(message)-1)
		for i := 1; i < len(message); i++ {
			keys[i-1] = string(message[i])
		}

		exists := store.Exists(keys...)
		return fmt.Appendf(nil, ":%d\r\n", exists), nil
	default:
		return fmt.Appendf(nil, "-ERR unknown command '%s'\r\n", command), nil
	}
}
