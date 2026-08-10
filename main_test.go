package main

import (
	"bytes"
	"net"
	"testing"
)

func Test_Server_Connects(t *testing.T) {
	conn, err := net.Dial("tcp", ":6379")

	if err != nil {
		t.Error("couldn't connect to server: ", err)
	}

	defer conn.Close()
}

func Test_Ping(t *testing.T) {
	conn, err := net.Dial("tcp", ":6379")

	if err != nil {
		t.Fatal("couldn't connect to Redis server: ", err)
	}

	defer conn.Close()

	pingPayload := []byte("*1\r\n$4\r\nPING\r\n")

	if _, err := conn.Write(pingPayload); err != nil {
		t.Fatal("could not write PING payload to Redis server")
	}

	out := make([]byte, 1024)
	expectedOut := []byte("+PONG\r\n")
	bytesRead, err := conn.Read(out)
	if err != nil {
		t.Fatal("coudln't read PING response from Redis server: ", err)
	}

	if !bytes.Equal(expectedOut, out[:bytesRead]) {
		t.Errorf("expected '%s', got '%s'\n", string(expectedOut), string(out))
	}
}
