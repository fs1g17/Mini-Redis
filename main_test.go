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

func startTestServer(kv *store.RedisStore) (string, error) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", testServerFailure
	}

	go func() {
		for {
			_conn, err := ln.Accept()
			if err != nil {
				return
			}

			go RedisConnection(_conn, kv)
		}
	}()

	return ln.Addr().String(), nil
}

func Test_MultipleClients(t *testing.T) {
	kv := store.NewRedisStore()

	addrStr, err := startTestServer(kv)
	if err != nil {
		t.Fatal("couldn't start test server: ", err)
	}

	conn1, err := net.Dial("tcp", addrStr)
	if err != nil {
		t.Fatal("couldn't connect to Redis server: ", err)
	}

	conn2, err := net.Dial("tcp", addrStr)
	if err != nil {
		t.Fatal("couldn't connect to Redis server: ", err)
	}

	_, err = conn1.Write([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"))
	if err != nil {
		t.Fatal("couldn't write SET foo bar to server")
	}

	want := []byte("+OK\r\n")
	out := make([]byte, 1024)
	bytesRead, err := conn1.Read(out)
	if !bytes.Equal(want, out[:bytesRead]) {
		t.Errorf("expected '%s', got '%s'\n", string(want), string(out[:bytesRead]))
	}

	_, err = conn2.Write([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"))
	if err != nil {
		t.Fatal("couldn't write GET foo to server")
	}

	want = []byte("$3\r\nbar\r\n")
	out = make([]byte, 1024)
	bytesRead, err = conn2.Read(out)
	if !bytes.Equal(want, out[:bytesRead]) {
		t.Errorf("expected '%s', got '%s'\n", string(want), string(out[:bytesRead]))
	}
}
