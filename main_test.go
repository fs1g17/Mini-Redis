package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"testing"
	"time"
)

var testServerFailure = errors.New("couldn't start test server")

func Test_RedisConnection(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	store := NewRedisStore()
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

	store := NewRedisStore()

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

/*
cases to test:
  - valid data
  - incomplete data
  - invalid data
  - negative length
  - incorrect byte length
*/
func Test_ParseData(t *testing.T) {
	t.Run("complete input", func(t *testing.T) {
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

			if !bytes.Equal(expectedBytes, out) {
				t.Fatalf("expected: %s, got: %s\n", string(expectedBytes), string(out))
			}

			if readBytes != expectedReadBytes {
				t.Fatalf("expected read bytes: %d, got: %d", expectedReadBytes, readBytes)
			}
		}
	})

	t.Run("incomplete input", func(t *testing.T) {
		type testStruct struct {
			input []byte
		}

		tests := make([]testStruct, 0, 10)

		str := "$5\r\nhello\r\n"
		for i := 0; i < len(str); i++ {
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

				if out != nil {
					t.Fatal("should have nil output")
				}
			})
		}
	})

	t.Run("negative length", func(t *testing.T) {
		input := []byte("$-2\r\nhello\r\n")
		out, readBytes, err := parseData(input)
		if err == nil {
			t.Fatal("should've got error")
		}

		if !errors.Is(err, invalidDataErr) {
			t.Fatalf("expected invalidDataErr, but got %v\n", err)
		}

		if readBytes != 0 {
			t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
		}

		if out != nil {
			t.Fatal("should have nil output")
		}
	})

	t.Run("non-digit length", func(t *testing.T) {
		input := []byte("$abc")
		out, readBytes, err := parseData(input)
		if err == nil {
			t.Fatal("should've got error")
		}

		if !errors.Is(err, invalidDataErr) {
			t.Fatalf("expected invalidDataErr, but got %v\n", err)
		}

		if readBytes != 0 {
			t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
		}

		if out != nil {
			t.Fatal("should have nil output")
		}
	})

	t.Run("invalid chars", func(t *testing.T) {
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

			if readBytes != 0 {
				t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
			}

			if out != nil {
				t.Fatal("should have nil output")
			}
		}
	})

	t.Run("missing length", func(t *testing.T) {
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

			if readBytes != 0 {
				t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
			}

			if out != nil {
				t.Fatal("should have nil output")
			}
		}
	})

	t.Run("data too short", func(t *testing.T) {
		// if it's too short - then we shouldn't get an error
		// we treat it as incomplete - can't rely on \r\n because data can contain itf
		input := []byte("$2\r\nh\r\n")
		out, readBytes, err := parseData(input)
		if err != nil {
			t.Fatalf("didn't expect error, but got %v\n", err)
		}

		if readBytes != 0 {
			t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
		}

		if out != nil {
			t.Fatal("should have nil output")
		}
	})

	t.Run("data too long", func(t *testing.T) {
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

		if readBytes != 0 {
			t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
		}

		if out != nil {
			t.Fatal("should have nil output")
		}
	})
}

/*
cases to test:
  - valid message
  - incomplete message
  - invalid message
  - negative length
  - incorrect byte length
*/
func Test_ParseMessage(t *testing.T) {
	t.Run("complete input", func(t *testing.T) {
		tests := []struct {
			input    []byte
			expected [][]byte
		}{
			{
				input:    []byte("*0\r\n"),
				expected: [][]byte{},
			},
			{
				input: []byte("*1\r\n$4\r\nPING\r\n"),
				expected: [][]byte{
					[]byte("PING"),
				},
			},
			{
				input: []byte("*1\r\n$4\r\nECHO\r\n"),
				expected: [][]byte{
					[]byte("ECHO"),
				},
			},
			{
				input: []byte("*2\r\n$4\r\nECHO\r\n$5\r\nhello\r\n"),
				expected: [][]byte{
					[]byte("ECHO"),
					[]byte("hello"),
				},
			},
		}

		for _, tt := range tests {
			out, readBytes, err := parseMessage(tt.input)
			if err != nil {
				t.Fatalf("got error: %v", err)
			}

			if readBytes != len(tt.input) {
				t.Fatalf("expected readBytes %d, got %d\n", len(tt.input), readBytes)
			}

			if !slices.EqualFunc(tt.expected, out, bytes.Equal) {
				t.Fatalf("expected: %q, got: %q\n", tt.expected, out)
			}
		}
	})

	t.Run("incomplete PING input", func(t *testing.T) {
		type testStruct struct {
			input []byte
		}

		tests := make([]testStruct, 0, 10)

		str := "*1\r\n$4\r\nPING\r\n"
		for i := 0; i < len(str); i++ {
			tests = append(tests, testStruct{input: []byte(str[:i])})
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("testing PING substring length: %d\n", len(tt.input)), func(t *testing.T) {
				out, readBytes, err := parseMessage(tt.input)
				if err != nil {
					t.Fatal("shouldn't have an error")
				}

				if readBytes != 0 {
					t.Fatal("should've read 0 bytes")
				}

				if out != nil {
					t.Fatal("should have nil output")
				}
			})
		}
	})

	t.Run("incomplete ECHO input", func(t *testing.T) {
		type testStruct struct {
			input []byte
		}

		tests := make([]testStruct, 0, 10)
		baseStr := "*2\r\n$4\r\nECHO\r\n"
		str := "$5\r\nhello\r\n"
		for i := 0; i < len(str); i++ {
			tests = append(tests, testStruct{input: []byte(baseStr + str[:i])})
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("testing ECHO substring length: %d\n", len(tt.input)), func(t *testing.T) {
				out, readBytes, err := parseMessage(tt.input)
				if err != nil {
					t.Fatal("shouldn't have an error")
				}

				if readBytes != 0 {
					t.Fatal("should've read 0 bytes")
				}

				if out != nil {
					t.Fatal("should have nil output")
				}
			})
		}
	})

	t.Run("negative length", func(t *testing.T) {
		input := []byte("*-2\r\n$4\r\nPING\r\n")
		out, readBytes, err := parseMessage(input)
		if err == nil {
			t.Fatal("should've got error")
		}

		if !errors.Is(err, invalidMessageErr) {
			t.Fatalf("expected invalidMessageErr, but got %v\n", err)
		}
		if readBytes != 0 {
			t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
		}

		if out != nil {
			t.Fatal("should have nil output")
		}
	})

	t.Run("non-digit length", func(t *testing.T) {
		input := []byte("*abc")
		out, readBytes, err := parseMessage(input)
		if err == nil {
			t.Fatal("should've got error")
		}

		if !errors.Is(err, invalidMessageErr) {
			t.Fatalf("expected invalidMessageErr, but got %v\n", err)
		}

		if readBytes != 0 {
			t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
		}

		if out != nil {
			t.Fatal("should have nil output")
		}
	})

	t.Run("invalid chars", func(t *testing.T) {
		tests := []struct {
			input []byte
		}{
			{
				input: []byte("*5\rx"),
			},
			{
				input: []byte("*5x"),
			},
		}

		for _, tt := range tests {
			out, readBytes, err := parseMessage(tt.input)
			if err == nil {
				t.Fatal("should've got error")
			}

			if !errors.Is(err, invalidMessageErr) {
				t.Fatalf("expected invalidMessageErr, but got %v\n", err)
			}

			if readBytes != 0 {
				t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
			}

			if out != nil {
				t.Fatal("should have nil output")
			}
		}
	})

	t.Run("message too short", func(t *testing.T) {
		// if it's too short - then we shouldn't get an error
		// we treat it as incomplete - can't rely on \r\n because data can contain itf
		input := []byte("*2\r\n$4\r\nECHO\r\n")
		out, readBytes, err := parseMessage(input)
		if err != nil {
			t.Fatalf("didn't expect error, but got %v\n", err)
		}

		if readBytes != 0 {
			t.Fatalf("expected readBytes to be 0, got %d\n", readBytes)
		}

		if out != nil {
			t.Fatal("should have nil output")
		}
	})

	t.Run("multiple messages", func(t *testing.T) {
		input := []byte("*1\r\n$4\r\nPING\r\n*1\r\n$4\r\nPING\r\n")
		expected := [][]byte{
			[]byte("PING"),
		}
		out, readBytes, err := parseMessage(input)

		if err != nil {
			t.Fatalf("didn't expect error, but got %v\n", err)
		}

		if readBytes != 14 {
			t.Fatalf("expected readBytes to be 14, got %d\n", readBytes)
		}

		if !slices.EqualFunc(expected, out, bytes.Equal) {
			t.Fatalf("expected: %q, got: %q\n", expected, out)
		}
	})
}
