package main

import (
	"bytes"
	"errors"
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
		message := make([]byte, 0)
		for len(message) == 0 {
			data := make([]byte, 1024)
			n, err := conn.Read(data)
			if err != nil {
				log.Printf("Got error reading: %v\n", err)
				return
			}
			log.Printf("data: '%s' \n", string(data))
			buff = append(buff, data[:n]...)
			message, err = parseMessage(buff)
			if err != nil {
				log.Printf("Got error parsing: %v\n", err)
				return
			}

			if len(message) > 0 {
				//reslice the buffer
				newBuff := make([]byte, 0, 1024)
				buff = append(newBuff, buff[len(message):]...)
			}
		}

		log.Printf("Read value: '%s'", string(message))
		log.Printf("Raw bytes: '%v'", message)

		response, err := getResponse(message)
		if err != nil {
			log.Printf("Got error writing: %v\n", err)
			return
		}

		log.Printf("Writing response: '%s'", string(response))

		_, err = conn.Write(response)
		if err != nil {
			log.Printf("Got error writing: %v\n", err)
			return
		}
	}
}

func parseMessage(buff []byte) ([]byte, error) {
	message := make([]byte, 0, 512)

	// we can find the first \n with buff.IndexByte(data, '\n'), saw on the internet
	if buff[0] == '*' {
		countEnd := bytes.IndexByte(buff, '\n')
		if countEnd == -1 {
			// incomlpete data, need more bytes
			return message, nil
		}

		// countEnd-1 because we need to stop before the \r
		count, err := strconv.Atoi(string(buff[1 : countEnd-1]))
		if err != nil {
			return message, invalidMessageErr // doesn't parse correctly
		}

		start := countEnd + 1 // we found the first \n, start of the data is after \n
		for range count {
			// instead of passing start, we can slice the buff
			data, readBytes, err := parseData(buff[start:])
			if err != nil {
				return message, invalidMessageErr // invalid data
			}
			if readBytes == 0 {
				return message, nil // incomplete data, need more bytes
			}

			message = append(message, data...)
			start += readBytes
		}
		return message, nil
	}

	return message, invalidMessageErr
}

// returns the read data and number of bytes read
// so we can start reading new line
func parseData(buff []byte) ([]byte, int, error) {
	data := make([]byte, 0, 512)
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

func getResponse(message []byte) ([]byte, error) {
	if string(message) == "PING" {
		return []byte("+PONG"), nil
	}

	return nil, noMatchingResponseErr
}
