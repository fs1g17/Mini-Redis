package main

import (
	"bytes"
	"errors"
	"fmt"
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

/*
cases to test:
  - valid data
  - incomplete data
  - invalid data
  - negative length
  - incorrect byte length
*/
func Test_ParseDataCompleteInput(t *testing.T) {
	tests := []struct {
		input          []byte
		expectedString string
	}{
		{
			input:          []byte("$5\r\nhello\r\n"),
			expectedString: "hello",
		},
		{
			input:          []byte("$0\r\n\r\n"),
			expectedString: "",
		},
	}

	for _, tt := range tests {
		expectedBytes := []byte(tt.expectedString)
		expectedReadBytes := len(tt.input)
		out, readBytes, err := parseData(tt.input)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}

		if !bytes.Equal(expectedBytes, out[:len(tt.expectedString)]) {
			t.Fatalf("expected: %s, got: %s\n", string(expectedBytes), string(out[:len(tt.expectedString)]))
		}

		if readBytes != expectedReadBytes {
			t.Fatalf("expected read bytes: %d, got: %d", expectedReadBytes, readBytes)
		}
	}
}

func Test_ParseDataIncompleteInput(t *testing.T) {
	type testStruct struct {
		input []byte
	}

	tests := make([]testStruct, 0, 10)

	str := "$5\r\nhello\r\n"
	for i := 0; i < len(str)-1; i++ {
		tests = append(tests, testStruct{input: []byte(str[:i])})
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("testing hello substring length: %d\n", len(tt.input)), func(t *testing.T) {
			out, readBytes, err := parseData(tt.input)
			if err != nil {
				t.Fatal("shouldn't have an error")
			}

			if readBytes != 0 {
				t.Fatal("should've read 0 bytes")
			}

			if len(out) != 0 {
				t.Fatal("should have empty output")
			}
		})
	}
}

func Test_ParseDataNegativeLength(t *testing.T) {
	input := []byte("$-2\r\nhello\r\n")
	out, readBytes, err := parseData(input)
	if err == nil {
		t.Fatal("should've got error")
	}

	if !errors.Is(err, invalidDataErr) {
		t.Fatalf("expected invalidDataErr, but got %v\n", err)
	}

	if len(out) != 0 {
		t.Fatalf("expected length of output to be 0, got: %d", len(out))
	}

	if readBytes != 0 {
		t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
	}
}

func Test_NonDigitLength(t *testing.T) {
	input := []byte("$abc")
	out, readBytes, err := parseData(input)
	if err == nil {
		t.Fatal("should've got error")
	}

	if !errors.Is(err, invalidDataErr) {
		t.Fatalf("expected invalidDataErr, but got %v\n", err)
	}

	if len(out) != 0 {
		t.Fatalf("expected length of output to be 0, got: %d", len(out))
	}

	if readBytes != 0 {
		t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
	}
}

func Test_InvalidChars(t *testing.T) {
	tests := []struct {
		input []byte
	}{
		{
			input: []byte("$5\rx"),
		},
		{
			input: []byte("$5x"),
		},
	}

	for _, tt := range tests {
		out, readBytes, err := parseData(tt.input)
		if err == nil {
			t.Fatal("should've got error")
		}

		if !errors.Is(err, invalidDataErr) {
			t.Fatalf("expected invalidDataErr, but got %v\n", err)
		}

		if len(out) != 0 {
			t.Fatalf("expected length of output to be 0, got: %d", len(out))
		}

		if readBytes != 0 {
			t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
		}
	}
}

func Test_ParseDataMissingLength(t *testing.T) {
	tests := []struct {
		input []byte
	}{
		{
			input: []byte("$\r\nhello\r\n"),
		},
		{
			input: []byte("$\n"),
		},
		{
			input: []byte("$\r"),
		},
	}

	for _, tt := range tests {
		out, readBytes, err := parseData(tt.input)
		if err == nil {
			t.Fatal("should've got error")
		}

		if !errors.Is(err, invalidDataErr) {
			t.Fatalf("expected invalidDataErr, but got %v\n", err)
		}

		if len(out) != 0 {
			t.Fatalf("expected length of output to be 0, got: %d", len(out))
		}

		if readBytes != 0 {
			t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
		}
	}
}

func Test_ParseDataTooShort(t *testing.T) {
	// if it's too short - then we shouldn't get an error
	// we treat it as incomplete - can't rely on \r\n because data can contain itf
	input := []byte("$2\r\nh\r\n")
	out, readBytes, err := parseData(input)
	if err != nil {
		t.Fatalf("didn't expect error, but got %v\n", err)
	}

	if len(out) != 0 {
		t.Fatalf("expected length of output to be 0, got: %d", len(out))
	}

	if readBytes != 0 {
		t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
	}
}

func Test_ParseDataTooLong(t *testing.T) {
	// if it's too long, then we don't land \r\n - but if we do because the data is like hel\r\n
	// then the rest of the byte slice will be treated like start of the next command
	// and it will be invalid
	input := []byte("$2\r\nhello\r\n")
	out, readBytes, err := parseData(input)
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, invalidDataErr) {
		t.Fatalf("expected invalidDataErr, but got: %v\n", err)
	}

	if len(out) != 0 {
		t.Fatalf("expected length of output to be 0, got: %d", len(out))
	}

	if readBytes != 0 {
		t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
	}
}
