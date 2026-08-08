package main

import (
	"errors"
	"log"
	"net"
)

var invalidMessageErr = errors.New("invalid message")

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
	bLen := 0
	for {
		message := make([]byte, 0, 512)
		messageComplete := false
		for messageComplete == false {
			data := make([]byte, 1024)
			n, err := conn.Read(data)
			if err != nil {
				log.Printf("Got error reading: %v\n", err)
				return
			}
			bLen += n
			buff = append(buff, data...)
			n, err = parseMessage(buff, &message)
			if err != nil {
				log.Printf("Got error parsing: %v\n", err)
				return
			}
			buff = buff[n:]
			bLen -= n
			if n > 0 {
				messageComplete = true
			}
		}

		log.Printf("Read value: '%s'", string(message))

		_, err := conn.Write(message)
		if err != nil {
			log.Printf("Got error writing: %v\n", err)
			return
		}
	}
}

func parseMessage(buff []byte, message *[]byte) (int, error) {
	log.Printf("got '%c'", buff[0])
	if buff[0] == '+' || buff[0] == '-' || buff[0] == ':' {
		for i, b := range buff {
			if b == '\n' {
				*message = append(*message, buff[:i]...)
				return i, nil
			}
		}
		return 0, invalidMessageErr
	}

	return 0, invalidMessageErr
}
