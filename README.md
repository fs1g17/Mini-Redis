# Mini-Redis — Technical Spec

A key-value server that speaks the real Redis wire protocol (RESP) over raw TCP, so you can drive it with the actual `redis-cli` and `redis-benchmark` tools. Learning project, ~2 days.

## 1. Goals & non-goals

**Goals**

- Work below the framework layer: raw `net.Listen`, byte-level protocol parsing, no HTTP framework.
- Get comfortable with Go's concurrency primitives on shared mutable state.
- Ship something concrete and verifiable — `redis-cli` connects and behaves as expected.

**Non-goals (explicitly out of scope)**

- Clustering, replication, pub/sub, transactions (`MULTI`/`EXEC`), Lua scripting.
- Full RESP3, full command set, exact Redis error-message parity.
- Being fast. Correctness and clarity first; optimize only as a labeled stretch goal.

## 2. Learning targets

These are the _point_ of the project — be deliberate about each, not incidental:

- Raw TCP with `net`, buffered I/O with `bufio.Reader`/`bufio.Writer`.
- RESP protocol parsing and serialization from bytes.
- Concurrency: one goroutine per connection, shared store behind `sync.RWMutex`.
- Key expiry via `context` + a background goroutine (and/or lazy expiry on read).
- Graceful shutdown with `signal.NotifyContext` and connection draining.
- Idiomatic errors: `fmt.Errorf("...: %w", err)`, `errors.Is/As`.
- Table-driven tests with `t.Run` subtests.

## 3. The RESP protocol (what you must parse)

RESP is line-based, terminated by `\r\n` (CRLF). You need to handle these five types. The five leading bytes:

| Type          | Prefix | Example encoding                     |
| ------------- | ------ | ------------------------------------ |
| Simple String | `+`    | `+OK\r\n`                            |
| Error         | `-`    | `-ERR unknown command\r\n`           |
| Integer       | `:`    | `:42\r\n`                            |
| Bulk String   | `$`    | `$5\r\nhello\r\n` (`$-1\r\n` = null) |
| Array         | `*`    | `*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n`   |

**Key fact:** clients send every command as an **Array of Bulk Strings**. So `SET foo bar` arrives on the wire as:

```
*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n
```

Your read path only needs to fully parse **arrays of bulk strings**. Your write path needs to _produce_ all five types depending on the reply. That asymmetry is worth noticing.

## 4. Functional requirements — commands

### Must-have (Day 1–2 core)

| Command             | Behavior          | Reply on success                     | Reply on error/miss                |
| ------------------- | ----------------- | ------------------------------------ | ---------------------------------- |
| `PING`              | liveness          | `+PONG` (or echo arg as bulk string) | —                                  |
| `ECHO <msg>`        | return the arg    | bulk string of `<msg>`               | arity error                        |
| `SET <k> <v>`       | store, no expiry  | `+OK`                                | arity error                        |
| `GET <k>`           | fetch             | bulk string of value                 | null bulk `$-1` if missing/expired |
| `DEL <k> [k...]`    | delete keys       | `:<count deleted>`                   | —                                  |
| `EXISTS <k> [k...]` | count existing    | `:<count>`                           | —                                  |
| `INCR <k>`          | atomic +1, init 0 | `:<new value>`                       | error if value not an integer      |
| `EXPIRE <k> <secs>` | set TTL           | `:1` set, `:0` no such key           | error if secs not int              |
| `TTL <k>`           | seconds left      | `:<n>`, `:-1` no TTL, `:-2` no key   | —                                  |

### Stretch (only if core is solid)

- `DECR`, `INCRBY`, `SETEX`, `SET k v EX <secs>` (parse options).
- `KEYS <pattern>` (glob), `FLUSHALL`.
- Persistence: snapshot store to disk on `SAVE`/shutdown, reload on boot (JSON is fine to start; append-only log is the more educational version).

## 5. Behavioral requirements & edge cases

- **Arity:** wrong argument count returns `-ERR wrong number of arguments for '<cmd>' command`.
- **Unknown command:** `-ERR unknown command '<name>'`.
- **Case-insensitive** command names (`set`, `SET`, `Set` all work). Keys/values are case-sensitive and binary-safe (bulk strings can contain any bytes, including `\r\n`).
- **Expiry semantics:** an expired key must behave as absent for `GET`/`EXISTS`/`TTL`. Decide and document your strategy — lazy (check-on-read) is simplest; a background sweeper is the concurrency exercise. Doing both is realistic.
- **INCR on non-numeric** value → `-ERR value is not an integer or out of range`.
- **Concurrency:** two clients hitting the same key must not corrupt state or race. `INCR` must be atomic. Run tests with `-race`.
- **Malformed input:** a bad RESP frame shouldn't crash the server — return an error reply or close that one connection, log it, keep serving others.

## 6. Architecture

Suggested package layout — small and consumer-driven, not Java-style layers:

```
mini-redis/
  cmd/miniredis/main.go   // wire everything up, flags, signal handling
  internal/server/        // TCP accept loop, per-conn goroutine
  internal/resp/          // RESP reader + writer (the parser)
  internal/store/         // KV store: Get/Set/Del/Incr/Expire + mutex
  internal/command/       // dispatch: []string -> store call -> reply
```

**Request lifecycle:**

1. `Accept()` a connection → spawn a goroutine.
2. Loop: `resp.ReadCommand(bufReader)` → `[]string` (or `[][]byte`).
3. `command.Dispatch(store, args)` → a reply value.
4. `resp.Write(bufWriter, reply)` → `Flush()`.
5. On EOF/error, close the conn and return from the goroutine.

**Concurrency model:** one goroutine per connection. All of them share one `*store.Store`, which guards its map with a `sync.RWMutex` (`RLock` for reads, `Lock` for writes). Keep the lock _inside_ the store's methods — callers shouldn't know it exists. Stretch: shard the map into N buckets by key hash to cut lock contention, and benchmark before/after.

## 7. Interfaces to aim for

Let the consumer define interfaces. For example the command layer shouldn't depend on a concrete store — it should depend on a small interface it defines:

```go
// in internal/command
type Store interface {
    Get(key string) (string, bool)
    Set(key, val string)
    Del(keys ...string) int
    // ...only what command actually calls
}
```

Accept interfaces, return structs. Keep them minimal.

## 8. Graceful shutdown (don't skip this)

- `ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`.
- On cancel: stop accepting new connections, close the listener, let in-flight commands finish (a `sync.WaitGroup` over connection goroutines with a timeout), then exit.
- The background expiry sweeper (if you build one) must also watch `ctx.Done()` and exit cleanly.

## 9. Configuration

- `-port` (default `6379`), `-addr` (default `127.0.0.1`).
- Optional `-snapshot <path>` for the persistence stretch goal.

## 10. Testing requirements

- **Unit test the RESP parser** hardest — table-driven: valid frames, partial frames, `$-1` null, empty array, oversized length, embedded CRLF in bulk strings.
- **Unit test the store**: expiry (use an injectable clock, not real `time.Sleep`), `INCR` on bad values, `DEL` counts.
- **Concurrency test**: N goroutines hammering `INCR` on one key; assert final value. Always run `go test -race ./...`.
- **Integration**: start the server on a random port in a test, connect with a raw `net.Dial`, send bytes, assert the reply. Bonus: run the real `redis-cli -p <port> ping` against it.

## 11. Acceptance criteria (definition of done)

- [ ] `redis-cli -p 6379 ping` → `PONG`.
- [ ] `SET foo bar` then `GET foo` → `"bar"`; `GET nope` → `(nil)`.
- [ ] `INCR counter` returns increasing integers across repeated calls and across two concurrent clients.
- [ ] `EXPIRE foo 1` then `GET foo` after 1s → `(nil)`; `TTL` reports correctly.
- [ ] `DEL`/`EXISTS` return correct counts.
- [ ] Unknown command and wrong arity return proper `-ERR` replies without crashing.
- [ ] `Ctrl-C` shuts down cleanly with no panic and no leaked goroutines.
- [ ] `go test -race ./...` passes.

## 12. Suggested milestones

**Day 1**

1. TCP echo server: accept, read bytes, write them back. Confirm `nc localhost 6379` works.
2. RESP reader: parse an array-of-bulk-strings into `[]string`. Unit test it.
3. RESP writer: simple string, error, integer, bulk string, null. Unit test it.
4. Wire `PING`/`ECHO` end to end. `redis-cli ping` → `PONG`. First real win.

**Day 2** 5. Store + `SET`/`GET`/`DEL`/`EXISTS` behind a mutex. 6. `INCR` (atomic) + concurrency test with `-race`. 7. `EXPIRE`/`TTL` + expiry strategy (lazy first, sweeper if time). 8. Graceful shutdown on `SIGINT`. 9. Polish: arity/unknown-command errors, README, `go test -race ./...` green. 10. (Stretch) persistence or map sharding + a `redis-benchmark` run.

## 13. Reference points to check yourself against

- Redis protocol spec: the RESP section of redis.io/docs — authoritative for the wire format.
- "Build Your Own Redis" (Build-Your-Own-X list) and CodeCrafters' Redis challenge — both good for checking your approach without copying.
- Go stdlib docs: `net`, `bufio`, `sync`, `context`, `os/signal`.
