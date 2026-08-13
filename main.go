package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
)

var invalidDataErr = errors.New("invalid data")
var invalidMessageErr = errors.New("invalid message")

func main() {
	ln, err := net.Listen("tcp", ":6379")

	if err != nil {
		// handle error
		log.Fatalf("Listen error: %s\n", err)
	}

	log.Printf("Listening on address: %s\n", ln.Addr().String())

	store := NewRedisStore()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			log.Printf("Got error on accepting connection: %v\n", err)
			continue
		}

		go RedisConnection(conn, store)
	}
}

func RedisConnection(conn net.Conn, store Store) {
	defer conn.Close()

	buff := make([]byte, 0, 1024)
	for {
		var message [][]byte
		for {
			// read buff first - in case more than 1 message came
			readMessage, bytesRead, err := parseMessage(buff)
			if err != nil {
				log.Printf("Got error parsing: %v\n", err)
				return
			}

			if bytesRead > 0 {
				//reslice the buffer
				newBuff := make([]byte, 0, 1024)
				buff = append(newBuff, buff[bytesRead:]...)
				message = readMessage
				break
			}

			data := make([]byte, 1024)
			n, err := conn.Read(data)
			if err != nil {
				log.Printf("Got error reading: %v\n", err)
				return
			}
			buff = append(buff, data[:n]...)
		}

		response, err := getResponse(message, store)
		if err != nil {
			log.Printf("Got error writing: %v\n", err)
			return
		}

		if response == nil {
			log.Printf("Got empty response, waiting for more bytes")
			continue
		}

		_, err = conn.Write(response)
		if err != nil {
			log.Printf("Got error writing: %v\n", err)
			return
		}
	}
}

func parseMessage(buff []byte) ([][]byte, int, error) {
	message := make([][]byte, 0, 512)
	if len(buff) == 0 {
		// incomplete data, need more bytes
		return nil, 0, nil
	}

	// we can find the first \n with buff.IndexByte(message, '\n'), saw on the internet
	if buff[0] == '*' {
		if len(buff) < 2 {
			// incomplete message
			return nil, 0, nil
		}

		i := 1
		for i = 1; i < len(buff); i++ {
			if buff[i] >= '0' && buff[i] <= '9' {
				// it's a digit
				continue
			} else if buff[i] == '\r' {
				break
			} else {
				return nil, 0, invalidMessageErr
			}
		}

		// position of \r
		lengthEnd := i
		// missing length: $\r\n
		if lengthEnd == 1 {
			return nil, 0, invalidMessageErr
		}

		// now we are at \r - need to validate next byte is \n
		i++
		if i >= len(buff) {
			// expecting LF, but buffer too small, incomplete message
			return nil, 0, nil
		}
		if buff[i] != '\n' {
			// expecting LF, but byte isn't LF
			return nil, 0, invalidMessageErr
		}

		count, err := strconv.Atoi(string(buff[1:lengthEnd]))
		if err != nil {
			return nil, 0, invalidMessageErr
		}

		if count < 0 {
			return nil, 0, invalidMessageErr
		}

		start := i + 1 // found \n, start of data is after

		// this is where we get
		for range count {
			if start >= len(buff) {
				// incomplete message, no point trying to read data
				return nil, 0, nil
			}

			// instead of passing start, we can slice the buff
			data, readBytes, err := parseData(buff[start:])
			if err != nil {
				return nil, 0, invalidMessageErr // invalid data
			}
			if readBytes == 0 {
				return nil, 0, nil // incomplete data, need more bytes
			}

			message = append(message, data)
			start += readBytes
		}

		return message, start, nil
	}

	return nil, 0, invalidMessageErr
}

// returns the read data and number of bytes read
// so we can start reading new line
func parseData(buff []byte) ([]byte, int, error) {
	data := make([]byte, 0, 512)
	if len(buff) == 0 {
		// incomplete data, need more bytes
		return nil, 0, nil
	}

	if buff[0] == '$' {
		if len(buff) < 2 {
			// incomplete data
			return nil, 0, nil
		}

		// ok so instead of all that we can read length until we get CR
		i := 1
		for i = 1; i < len(buff); i++ {
			if buff[i] >= '0' && buff[i] <= '9' {
				// it is a digit
				continue
			} else if buff[i] == '\r' {
				break
			} else {
				return nil, 0, invalidDataErr
			}
		}

		// position of \r
		lengthEnd := i
		// missing length: $\r\n
		if lengthEnd == 1 {
			return nil, 0, invalidDataErr
		}

		// now we are at \r - need to validate next byte is \n
		i++
		if i >= len(buff) {
			// expecting LF, but buffer too small, incomplete data
			return nil, 0, nil
		}
		if buff[i] != '\n' {
			return nil, 0, invalidDataErr
		}

		length, err := strconv.Atoi(string(buff[1:lengthEnd]))
		if err != nil {
			return nil, 0, invalidDataErr
		}

		if length < 0 {
			return nil, 0, invalidDataErr
		}

		// check if there's enough bytes to read
		// length end is \n
		// data starts at lengthEnd+2 - the entire length of the data, then \r\n - which is +2
		// so it's lengthEnd + 2 + length + 2 - should take us to \r\n of the data
		// which means we simplify it to lengthEnd + length + 4
		if len(buff) < lengthEnd+length+4 {
			return nil, 0, nil // if the data is incomplete -> message incomplete
		}

		// if after the data we don't get \r\n, then data is malformed
		// i.e it could be longer than specified - e.g. $2\r\nhi\r\n
		if buff[lengthEnd+length+2] != '\r' || buff[lengthEnd+length+3] != '\n' {
			return nil, 0, invalidDataErr
		}

		// we only read the data, but we "consume" the \r\n as well
		for i := lengthEnd + 2; i < lengthEnd+2+length; i++ {
			data = append(data, buff[i])
		}

		// last byte consumed index is: lengthEnd+2+length+2 = lengthEnd+length+4
		return data, lengthEnd + length + 4, nil
	}

	return nil, 0, invalidDataErr
}

func getResponse(message [][]byte, store Store) ([]byte, error) {
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
