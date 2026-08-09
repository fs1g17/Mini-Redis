package main

import (
	"errors"
	"log"
	"net"
)

var invalidMessageErr = errors.New("invalid message")
var invalidDataErr = errors.New("invalid data")

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

		_, err := conn.Write(message)
		if err != nil {
			log.Printf("Got error writing: %v\n", err)
			return
		}
	}
}

func parseMessage(buff []byte) ([]byte, error) {
	message := make([]byte, 0, 512)
	if buff[0] == '*' {
		count := int(buff[1] - '0')
		start := 4 // omit the *,1,\r,\n, bytes
		for range count {
			data, readBytes, err := parseData(buff, start)
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
func parseData(buff []byte, start int) ([]byte, int, error) {
	data := make([]byte, 0, 512)
	if buff[start] == '$' {
		length := int(buff[start+1] - '0')
		// check if there are enough bytes to read
		// $4\r\n - that's 4 bytes, then the end \r\n is the last 2, so 6 total
		if len(buff) < start+6+length {
			return data, 0, nil // if the data is incomplete -> message incomplete
		}

		// the 4th byte is the start of the data
		// we only read the data, but we "consume" the \r\n as well
		for i := start + 4; i < start+4+length; i++ {
			data = append(data, buff[i])
		}

		// return data, bytes consumed is length of data + 6
		// because $, 4, \r, \n, \r, \n
		return data, length + 6, nil
	}

	return data, 0, invalidDataErr
}
