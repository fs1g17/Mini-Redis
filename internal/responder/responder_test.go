package responder

import (
	"bytes"
	"testing"

	"github.com/fs1g17/Mini-Redis/internal/store"
)

func Test_Response(t *testing.T) {
	t.Run("test PING response", func(t *testing.T) {
		store := store.NewRedisStore()

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
			got, err := GetResponse(tt.message, store)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test ECHO response", func(t *testing.T) {
		store := store.NewRedisStore()

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
			got, err := GetResponse(tt.message, store)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test SET response", func(t *testing.T) {
		store := store.NewRedisStore()

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
			got, err := GetResponse(tt.message, store)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test GET response", func(t *testing.T) {
		store := store.NewRedisStore()

		store.Set("foo", "bar")

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
			got, err := GetResponse(tt.message, store)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test DEL response", func(t *testing.T) {
		store := store.NewRedisStore()

		store.Set("1", "1")
		store.Set("2", "2")
		store.Set("3", "3")

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
			got, err := GetResponse(tt.message, store)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test EXISTS response", func(t *testing.T) {
		store := store.NewRedisStore()

		store.Set("1", "1")
		store.Set("2", "2")
		store.Set("3", "3")

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
			got, err := GetResponse(tt.message, store)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test EXPIRE reponse", func(t *testing.T) {
		store := store.NewRedisStore()

		store.Set("foo", "bar")
		store.Set("foo2", "bar2")

		tests := []struct {
			message [][]byte
			want    []byte
		}{
			{
				message: [][]byte{
					[]byte("EXPIRE"),
				},
				want: []byte("-ERR wrong number of arguments for 'expire' command\r\n"),
			},
			{
				message: [][]byte{
					[]byte("EXPIRE"),
					[]byte("foo"),
					[]byte("2"),
				},
				want: []byte(":1\r\n"),
			},
			{
				message: [][]byte{
					[]byte("EXPIRE"),
					[]byte("foo2"),
					[]byte("bar"),
				},
				want: []byte("-ERR seconds must be positive int\r\n"),
			},
			{
				message: [][]byte{
					[]byte("EXPIRE"),
					[]byte("foo2"),
					[]byte("-2"),
				},
				want: []byte("-ERR seconds must be positive int\r\n"),
			},
		}

		for _, tt := range tests {
			got, err := GetResponse(tt.message, store)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test INCR response", func(t *testing.T) {
		store := store.NewRedisStore()

		store.Set("foo", "12")
		store.Set("foo2", "bar2")

		tests := []struct {
			message [][]byte
			want    []byte
		}{
			{
				message: [][]byte{
					[]byte("INCR"),
				},
				want: []byte("-ERR wrong number of arguments for 'incr' command\r\n"),
			},
			{
				message: [][]byte{
					[]byte("INCR"),
					[]byte("new"),
				},
				want: []byte(":1\r\n"),
			},
			{
				message: [][]byte{
					[]byte("INCR"),
					[]byte("foo"),
				},
				want: []byte(":13\r\n"),
			},
			{
				message: [][]byte{
					[]byte("INCR"),
					[]byte("foo2"),
				},
				want: []byte("-ERR value is not an integer or out of range\r\n"),
			},
		}

		for _, tt := range tests {
			got, err := GetResponse(tt.message, store)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})

	t.Run("test unknown command response", func(t *testing.T) {
		store := store.NewRedisStore()

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
			got, err := GetResponse(tt.message, store)
			if err != nil {
				t.Errorf("didn't expect error, but got: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		}
	})
}
