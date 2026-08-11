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

func Test_Funcs(t *testing.T) {
	tests := []struct {
		command string
		payload []byte
		want    []byte
	}{
		{
			command: "PING",
			payload: []byte("*1\r\n$4\r\nPING\r\n"),
			want:    []byte("+PONG\r\n"),
		},
		{
			command: "ECHO",
			payload: []byte("*2\r\n$4\r\nECHO\r\n$5\r\nhello\r\n"),
			want:    []byte("$5\r\nhello\r\n"),
		},
		{
			command: "ECHO",
			payload: []byte("*2\r\n$4\r\nECHO\r\n$11\r\nhello world\r\n"),
			want:    []byte("$11\r\nhello world\r\n"),
		},
		{
			command: "ECHO",
			payload: []byte("*3\r\n$4\r\nECHO\r\n$5\r\nhello\r\n$5\r\nworld\r\n"),
			want:    []byte("-ERR wrong number of arguments for 'echo' command\r\n"),
		},
		{
			command: "ECHO",
			payload: []byte("*1\r\n$4\r\nECHO\r\n"),
			want:    []byte("-ERR wrong number of arguments for 'echo' command\r\n"),
		},
	}

	for _, tt := range tests {
		addrStr, err := startTestServer()
		if err != nil {
			t.Fatal("couldn't start test server: ", err)
		}

		conn, err := net.Dial("tcp", addrStr)

		if err != nil {
			t.Fatal("couldn't connect to Redis server: ", err)
		}

		defer conn.Close()

		if _, err := conn.Write(tt.payload); err != nil {
			t.Fatalf("could not write %s payload to Redis server", tt.command)
		}

		out := make([]byte, 1024)
		bytesRead, err := conn.Read(out)
		if !bytes.Equal(tt.want, out[:bytesRead]) {
			t.Errorf("expected '%s', got '%s'\n", string(tt.want), string(out[:bytesRead]))
		}
	}
}
