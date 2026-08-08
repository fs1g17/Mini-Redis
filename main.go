package main

import (
	"log"
	"net"
)

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

	for {
		command := make([]byte, 1024)
		n, err := conn.Read(command)
		if err != nil {
			log.Printf("Got error reading: %v\n", err)
			return
		}
		log.Printf("Read %d bytes\n", n)
		log.Printf("Read value: %s", string(command))

		n, err = conn.Write(command)
		if err != nil {
			log.Printf("Got error writing: %v\n", err)
			return
		}
	}
}
