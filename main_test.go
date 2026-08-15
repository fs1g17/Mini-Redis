package main

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/fs1g17/Mini-Redis/internal/store"
)

//TODO: add tests for 1 store, multiple clients

var testServerFailure = errors.New("couldn't start test server")

func Test_RedisConnection(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	store := store.NewRedisStore()
	go RedisConnection(server, store)

	if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	if _, err := client.Write([]byte("*1\r\n$4\r\nPING\r\n*1\r\n$4\r\nPING\r\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	want := []byte("+PONG\r\n+PONG\r\n")
	got := make([]byte, len(want))
	if _, err := io.ReadFull(client, got); err != nil {
		t.Fatalf("read: %v (got %q so far)", err, got[:])
	}

	if !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func startTestServer() (string, error) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", testServerFailure
	}

	store := store.NewRedisStore()

	go func() {
		_conn, err := ln.Accept()
		if err != nil {
			return
		}

		go RedisConnection(_conn, store)
	}()

	return ln.Addr().String(), nil
}

func Test_SimpleFuncs(t *testing.T) {
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
