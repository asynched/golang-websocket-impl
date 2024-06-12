# golang-websocket-impl

This is an implementation of the websocket protocol in Go. It follows the original [RFC 6455](https://datatracker.ietf.org/doc/html/rfc6455) specification (not entirely but mostly).

It uses the `net/http` package as the HTTP server and builds on top of it.

## How to run

Run the server with

```go
go run main.go
```

Run a sample client with

```bash
bun run scripts/test.js
```
