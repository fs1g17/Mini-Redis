package responder

import (
	"bytes"
	"testing"
)

type FakeStore struct {
	data map[string]string
}

func (fs *FakeStore) Del(keys ...string) int {
	count := 0
	for _, key := range keys {
		if _, ok := fs.data[key]; !ok {
			continue
		}

		delete(fs.data, key)
		count++
	}
	return count
}

func (fs *FakeStore) Exists(keys ...string) int {
	count := 0
	for _, key := range keys {
		if _, ok := fs.data[key]; !ok {
			continue
		}
		count++
	}
	return count
}

func (fs *FakeStore) Get(key string) (string, bool) {
	value, ok := fs.data[key]
	return value, ok
}

func (fs *FakeStore) Set(key string, value string) {
	fs.data[key] = value
}

func Test_Response(t *testing.T) {
	t.Run("test PING response", func(t *testing.T) {
		fakeStore := &FakeStore{
			data: make(map[string]string),
		}

		tests := []struct {
			message [][]byte
			want    []byte
		}{
			{
				message: [][]byte{
					[]byte("PING"),
				},
				want: []byte("+PONG\r\n"),
			},
		}

		for _, tt := range tests {
			got, err := GetResponse(tt.message, fakeStore)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test ECHO response", func(t *testing.T) {
		fakeStore := &FakeStore{
			data: make(map[string]string),
		}

		tests := []struct {
			message [][]byte
			want    []byte
		}{
			{
				message: [][]byte{
					[]byte("ECHO"),
				},
				want: []byte("-ERR wrong number of arguments for 'echo' command\r\n"),
			},
			{
				message: [][]byte{
					[]byte("ECHO"),
					[]byte("hello"),
				},
				want: []byte("$5\r\nhello\r\n"),
			},
			{
				message: [][]byte{
					[]byte("ECHO"),
					[]byte("hello"),
					[]byte("world"),
				},
				want: []byte("-ERR wrong number of arguments for 'echo' command\r\n"),
			},
		}

		for _, tt := range tests {
			got, err := GetResponse(tt.message, fakeStore)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test SET response", func(t *testing.T) {
		fakeStore := &FakeStore{
			data: make(map[string]string),
		}

		tests := []struct {
			message [][]byte
			want    []byte
		}{
			{
				message: [][]byte{
					[]byte("SET"),
				},
				want: []byte("-ERR wrong number of arguments for 'set' command\r\n"),
			},
			{
				message: [][]byte{
					[]byte("SET"),
					[]byte("foo"),
					[]byte("bar"),
				},
				want: []byte("+OK\r\n"),
			},
			{
				message: [][]byte{
					[]byte("SET"),
					[]byte("foo"),
					[]byte("bar"),
					[]byte("car"),
				},
				want: []byte("-ERR wrong number of arguments for 'set' command\r\n"),
			},
		}

		for _, tt := range tests {
			got, err := GetResponse(tt.message, fakeStore)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test GET response", func(t *testing.T) {
		fakeStore := &FakeStore{
			data: make(map[string]string),
		}

		fakeStore.data["foo"] = "bar"

		tests := []struct {
			message [][]byte
			want    []byte
		}{
			{
				message: [][]byte{
					[]byte("GET"),
				},
				want: []byte("-ERR wrong number of arguments for 'get' command\r\n"),
			},
			{
				message: [][]byte{
					[]byte("GET"),
					[]byte("foo"),
				},
				want: []byte("$3\r\nbar\r\n"),
			},
			{
				message: [][]byte{
					[]byte("GET"),
					[]byte("car"),
				},
				want: []byte("$-1\r\n"),
			},
			{
				message: [][]byte{
					[]byte("GET"),
					[]byte("foo"),
					[]byte("bar"),
				},
				want: []byte("-ERR wrong number of arguments for 'get' command\r\n"),
			},
		}

		for _, tt := range tests {
			got, err := GetResponse(tt.message, fakeStore)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test DEL response", func(t *testing.T) {
		fakeStore := &FakeStore{
			data: make(map[string]string),
		}

		fakeStore.data["1"] = "1"
		fakeStore.data["2"] = "2"
		fakeStore.data["3"] = "3"

		tests := []struct {
			message [][]byte
			want    []byte
		}{
			{
				message: [][]byte{
					[]byte("DEL"),
				},
				want: []byte("-ERR wrong number of arguments for 'del' command\r\n"),
			},
			{
				message: [][]byte{
					[]byte("DEL"),
					[]byte("1"),
					[]byte("2"),
				},
				want: []byte(":2\r\n"),
			},
			{
				message: [][]byte{
					[]byte("DEL"),
					[]byte("3"),
				},
				want: []byte(":1\r\n"),
			},
			{
				message: [][]byte{
					[]byte("DEL"),
					[]byte("4"),
				},
				want: []byte(":0\r\n"),
			},
		}

		for _, tt := range tests {
			got, err := GetResponse(tt.message, fakeStore)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test EXISTS response", func(t *testing.T) {
		fakeStore := &FakeStore{
			data: make(map[string]string),
		}

		fakeStore.data["1"] = "1"
		fakeStore.data["2"] = "2"
		fakeStore.data["3"] = "3"

		tests := []struct {
			message [][]byte
			want    []byte
		}{
			{
				message: [][]byte{
					[]byte("EXISTS"),
				},
				want: []byte("-ERR wrong number of arguments for 'exists' command\r\n"),
			},
			{
				message: [][]byte{
					[]byte("EXISTS"),
					[]byte("1"),
					[]byte("2"),
				},
				want: []byte(":2\r\n"),
			},
			{
				message: [][]byte{
					[]byte("EXISTS"),
					[]byte("3"),
				},
				want: []byte(":1\r\n"),
			},
			{
				message: [][]byte{
					[]byte("EXISTS"),
					[]byte("4"),
				},
				want: []byte(":0\r\n"),
			},
		}

		for _, tt := range tests {
			got, err := GetResponse(tt.message, fakeStore)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test unknown command response", func(t *testing.T) {
		fakeStore := &FakeStore{
			data: make(map[string]string),
		}

		tests := []struct {
			message [][]byte
			want    []byte
		}{
			{
				message: [][]byte{
					[]byte("DO_SOMETHING"),
				},
				want: []byte("-ERR unknown command 'DO_SOMETHING'\r\n"),
			},
		}

		for _, tt := range tests {
			got, err := GetResponse(tt.message, fakeStore)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})
}
