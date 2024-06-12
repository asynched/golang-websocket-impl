package ws

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
)

const (
	opCodeContinuation = 0x0
	opCodeText         = 0x1
	opCodeBinary       = 0x2
	// 0x3 - 0x7 reserved
	opCodeClose = 0x8
	opCodePing  = 0x9
	opCodePong  = 0xA
	// 0xB - 0xF reserved
)

const magicWebsocketGUID string = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Conn is an interface that represents a connection
// that can be used to read and write data.
type Conn interface {
	// Write writes data to the connection.
	Write([]byte) (int, error)
	// Read reads data from the connection.
	Read([]byte) (int, error)
	// Close closes the connection.
	Close() error
}

type connImpl struct {
	conn   net.Conn
	rw     *bufio.ReadWriter
	buffer []byte
}

// Upgrades an HTTP connection to handle websocket communication.
// This function will return a Conn interface that can be used to read
// and write data, it adheres to the io.Reader and io.Writer interfaces.
func Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error) {
	h := r.Header

	if h.Get("Connection") != "Upgrade" {
		return nil, errors.New("missing 'connection' header")
	}

	if h.Get("Upgrade") != "websocket" {
		return nil, errors.New("missing 'upgrade' header")
	}

	if h.Get("Sec-WebSocket-Version") != "13" {
		return nil, errors.New("invalid version")
	}

	if h.Get("Sec-WebSocket-Key") == "" {
		return nil, errors.New("missing 'sec-websocket-key' header")
	}

	key := h.Get("Sec-WebSocket-Key")

	secKey := hashKey(key)

	w.Header().Set("Upgrade", "websocket")
	w.Header().Set("Connection", "Upgrade")
	w.Header().Set("Sec-WebSocket-Accept", secKey)
	w.WriteHeader(http.StatusSwitchingProtocols)

	conn, rw, err := http.NewResponseController(w).Hijack()

	if err != nil {
		return nil, err
	}

	return &connImpl{
		conn:   conn,
		rw:     rw,
		buffer: nil,
	}, nil
}

func (c *connImpl) Write(p []byte) (int, error) {
	frame := getDataFrame(len(p))

	_, err := c.rw.Write(frame)

	if err != nil {
		return 0, err
	}

	n, err := c.rw.Write(p)

	if err != nil {
		return 0, err
	}

	err = c.rw.Flush()

	return n, err
}

func (c *connImpl) Read(p []byte) (int, error) {
	if c.buffer != nil {
		n := copy(p, c.buffer)

		if n == len(c.buffer) {
			c.buffer = nil
		} else {
			c.buffer = c.buffer[n:]
		}

		return n, nil
	}

	for {
		frame := make([]byte, 2)

		_, err := c.rw.Read(frame)

		if err != nil {
			return 0, err
		}

		opCode := frame[0] & 0x0F
		payloadLength := int(frame[1] & 0x7F)

		switch opCode {
		case opCodeText, opCodeBinary:
			if payloadLength == 0x7E {
				payload := make([]byte, 2)

				_, err = c.rw.Read(payload)

				if err != nil {
					return 0, err
				}

				payloadLength = int(payload[0])<<0x08 | int(payload[1])
			} else if payloadLength == 0x7F {
				payload := make([]byte, 8)

				_, err = c.rw.Read(payload)

				if err != nil {
					return 0, err
				}

				payloadLength = int(payload[0])<<0x38 | int(payload[1])<<0x30 | int(payload[2])<<0x28 | int(payload[3])<<0x20 | int(payload[4])<<0x18 | int(payload[5])<<0x10 | int(payload[6])<<0x08 | int(payload[7])
			}

			mask := make([]byte, 4)

			_, err = c.rw.Read(mask)

			if err != nil {
				return 0, err
			}

			payload := make([]byte, payloadLength)

			_, err = c.rw.Read(payload)

			if err != nil {
				return 0, err
			}

			for i := 0; i < len(payload); i++ {
				payload[i] ^= mask[i%4]
			}

			if payloadLength > len(p) {
				n := copy(p, payload)

				c.buffer = payload[n:]

				return n, nil
			}

			copy(p, payload)

			return len(payload), nil
		case opCodePing, opCodePong:
			// TODO: Handle ping and pong later.
		case opCodeClose:
			return 0, errors.New("connection closed")
		default:
			return 0, errors.New("unknown opcode")
		}
	}
}

func (c *connImpl) Close() error {
	return c.conn.Close()
}

// hashKey hashes a key using the SHA1 algorithm and returns the base64 encoded result.
// It is required to hash the key provided by the client and append a predefined GUID
// to it before encoding it to base64. This comes from the original WebSocket spec.
func hashKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key + magicWebsocketGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// getDataFrame returns the beginning of a WebSocket frame with the given size.
func getDataFrame(size int) []byte {
	buffer := make([]byte, 0)

	buffer = append(buffer, 0x80|byte(opCodeText)&0x0F)

	if size <= 125 {
		buffer = append(buffer, byte(size))
	} else if size <= 65535 {
		buffer = append(buffer, 126)
		buffer = append(buffer, byte(size>>0x08))
		buffer = append(buffer, byte(size&0xFF))
	} else {
		buffer = append(buffer, 127)
		buffer = append(buffer, byte(size>>0x38))
		buffer = append(buffer, byte(size>>0x30))
		buffer = append(buffer, byte(size>>0x28))
		buffer = append(buffer, byte(size>>0x20))
		buffer = append(buffer, byte(size>>0x18))
		buffer = append(buffer, byte(size>>0x10))
		buffer = append(buffer, byte(size>>0x08))
		buffer = append(buffer, byte(size&0xFF))
	}

	return buffer
}
