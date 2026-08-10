package main

import (
	"bytes"
	"errors"
	"net"
	"testing"
)

var testServerFailure = errors.New("couldn't start test server")

func startTestServer() (string, error) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", testServerFailure
	}

	go func() {
		_conn, err := ln.Accept()
		if err != nil {
			return
		}

		go RedisConnection(_conn)
	}()

	return ln.Addr().String(), nil
}

func Test_Ping(t *testing.T) {
	addrStr, err := startTestServer()
	if err != nil {
		t.Fatal("couldn't start test server: ", err)
	}

	conn, err := net.Dial("tcp", addrStr)

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
		t.Errorf("expected '%s', got '%s'\n", string(expectedOut), string(out[:bytesRead]))
	}
}

func Test_Echo(t *testing.T) {
	addrStr, err := startTestServer()
	if err != nil {
		t.Fatal("couldn't start test server: ", err)
	}

	conn, err := net.Dial("tcp", addrStr)

	if err != nil {
		t.Fatal("couldn't connect to Redis server: ", err)
	}

	defer conn.Close()

	echoPayload := []byte("*2\r\n$4\r\nECHO\r\n$5\r\nhello\r\n")

	if _, err := conn.Write(echoPayload); err != nil {
		t.Fatal("could not write ECHO payload to Redis server")
	}

	out := make([]byte, 1024)
	expectedOut := []byte("+hello\r\n")
	bytesRead, err := conn.Read(out)
	if err != nil {
		t.Fatal("couldn't read ECHO response from Redis server: ", err)
	}

	if !bytes.Equal(expectedOut, out[:bytesRead]) {
		t.Errorf("expected '%s', got '%s'\n", string(expectedOut), string(out[:bytesRead]))
	}
}
