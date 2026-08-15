package main

import (
	"log"
	"net"

	"github.com/fs1g17/Mini-Redis/internal/parser"
	"github.com/fs1g17/Mini-Redis/internal/responder"
	"github.com/fs1g17/Mini-Redis/internal/store"
)

func main() {
	ln, err := net.Listen("tcp", ":6379")

	if err != nil {
		// handle error
		log.Fatalf("Listen error: %s\n", err)
	}

	log.Printf("Listening on address: %s\n", ln.Addr().String())

	kv := store.NewRedisStore()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			log.Printf("Got error on accepting connection: %v\n", err)
			continue
		}

		go RedisConnection(conn, kv)
	}
}

func RedisConnection(conn net.Conn, kv responder.Store) {
	defer conn.Close()

	buff := make([]byte, 0, 1024)
	for {
		var message [][]byte
		for {
			// read buff first - in case more than 1 message came
			readMessage, bytesRead, err := parser.ParseMessage(buff)
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

		response, err := responder.GetResponse(message, kv)
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
