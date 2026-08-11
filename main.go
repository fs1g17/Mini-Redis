package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
)

var invalidMessageErr = errors.New("invalid message")
var invalidDataErr = errors.New("invalid data")
var noMatchingResponseErr = errors.New("no matching response for message")

func main() {
	ln, err := net.Listen("tcp", ":6379")

	if err != nil {
		// handle error
		log.Fatalf("Listen error: %s\n", err)
	}

	log.Printf("Listening on address: %s\n", ln.Addr().String())

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			log.Printf("Got error on accepting connection: %v\n", err)
			continue
		}

		go RedisConnection(conn)
	}
}

func RedisConnection(conn net.Conn) {
	defer conn.Close()

	buff := make([]byte, 0, 1024)
	for {
		message := make([][]byte, 0)
		completedMessage := false
		for completedMessage == false {
			data := make([]byte, 1024)
			n, err := conn.Read(data)
			if err != nil {
				log.Printf("Got error reading: %v\n", err)
				return
			}
			log.Printf("data: '%s' \n", string(data))
			buff = append(buff, data[:n]...)
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
				completedMessage = true
			}

			log.Println("[")
			for i := range len(readMessage) {
				log.Println(string(readMessage[i]))
			}
			log.Println("]")

			log.Printf("Bytes read: '%d'", bytesRead)
		}

		log.Printf("Raw bytes: '%v'", message)

		response, err := getResponse(message)
		if err != nil {
			log.Printf("Got error writing: %v\n", err)
			return
		}

		if response == nil {
			log.Printf("Got empty response, waiting for more bytes")
			continue
		}

		log.Printf("Writing response: '%s'", string(response))

		_, err = conn.Write(response)
		if err != nil {
			log.Printf("Got error writing: %v\n", err)
			return
		}
	}
}

func parseMessage(buff []byte) ([][]byte, int, error) {
	message := make([][]byte, 0, 512)

	// we can find the first \n with buff.IndexByte(data, '\n'), saw on the internet
	if buff[0] == '*' {
		countEnd := bytes.IndexByte(buff, '\n')
		if countEnd == -1 {
			// incomlpete data, need more bytes
			return message, 0, nil
		}

		// countEnd-1 because we need to stop before the \r
		count, err := strconv.Atoi(string(buff[1 : countEnd-1]))
		if err != nil {
			return message, 0, invalidMessageErr // doesn't parse correctly
		}

		start := countEnd + 1 // we found the first \n, start of the data is after \n

		// this is where we get
		for range count {
			// instead of passing start, we can slice the buff
			data, readBytes, err := parseData(buff[start:])
			if err != nil {
				return message, 0, invalidMessageErr // invalid data
			}
			if readBytes == 0 {
				return message, 0, nil // incomplete data, need more bytes
			}

			message = append(message, data)
			start += readBytes
		}

		return message, start, nil
	}

	return message, 0, invalidMessageErr
}

// returns the read data and number of bytes read
// so we can start reading new line
func parseData(buff []byte) ([]byte, int, error) {
	data := make([]byte, 0, 512)
	if len(buff) == 0 {
		// incomplete data, need more bytes
		return data, 0, nil
	}

	if buff[0] == '$' {
		lengthEnd := bytes.IndexByte(buff, '\n')
		if lengthEnd == -1 {
			// incomplete data, need more bytes
			return data, 0, nil
		}

		length, err := strconv.Atoi(string(buff[1 : lengthEnd-1]))
		if err != nil {
			return data, 0, invalidDataErr
		}

		if length < 0 {
			return data, 0, invalidDataErr
		}

		// check if there's enough bytes to read
		// length end is \n
		// data starts at lengthEnd+1 - the entire length of the data, then \r\n - which is +2
		// so it's lengthEnd + 1 + length + 2 - should take us to \r\n of the data
		// which means we simplify it to lengthEnd + length + 3
		if len(buff) < lengthEnd+length+3 {
			return data, 0, nil // if the data is incomplete -> message incomplete
		}
		// we only read the data, but we "consume" the \r\n as well
		for i := lengthEnd + 1; i < lengthEnd+1+length; i++ {
			data = append(data, buff[i])
		}

		// last byte consumed index is: lengthEnd+1+length+2 = lengthEnd+length+3
		return data, lengthEnd + length + 3, nil
	}

	return data, 0, invalidDataErr
}

func getResponse(message [][]byte) ([]byte, error) {
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
	}

	return nil, noMatchingResponseErr
}
