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
			log.Printf("Got error: %v\n", err)
			continue
		}

		poop := make([]byte, 1024)

		n, err := conn.Read(poop)
		if err != nil {
			log.Printf("Got error reading: %v\n", err)
			continue
		}
		log.Printf("Read %d bytes\n", n)
		log.Printf("Read value: %s", string(poop))

		// so i need to parse the request and get just the body

		n, err = conn.Write(poop)
		if err != nil {
			log.Printf("Got error writing: %v\n", err)
			continue
		}

		conn.Close()
	}
}
