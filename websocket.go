package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Constants related to our websocket
const (
	bufferSize         = 200
	closedErrorMessage = "use of closed network connection"
)

// Custom error messages for websocket
var (
	// ErrAlreadyConnected is where a client attempted to connect when we are already connected
	ErrAlreadyConnected = errors.New("already connected")
)

// BasicWebsocket is something
type BasicWebsocket struct {
	conn      *websocket.Conn
	connected bool
	connMutex sync.Mutex

	url    string
	header http.Header

	readerMessages chan []byte
	readerDone     chan bool
	writerMessages chan []byte
	writerDone     chan bool

	// Time in between being disconnected and reconnecting
	ReconnectTime time.Duration
	// Whether the websocket should try to reconnect after getting disconnected
	AutoReconnect bool
	// Callback function to be called on websocket connect/reconnect
	OnConnect func()
	// Callback function to be called on every message received
	OnMessage func(b []byte) error
	// Callback function to be called on errors
	OnError func(err error)
}

// NewBasicWebsocket creates a new BasicWebsocket with a URL and Header
func NewBasicWebsocket(url string, header http.Header) *BasicWebsocket {
	return &BasicWebsocket{
		conn:      nil,
		connected: false,

		url:    url,
		header: header,

		readerMessages: make(chan []byte, bufferSize),
		readerDone:     make(chan bool),
		writerMessages: make(chan []byte, bufferSize),
		writerDone:     make(chan bool),

		ReconnectTime: 0,
		AutoReconnect: false,
		OnConnect:     func() {},
		OnMessage:     func(b []byte) error { return nil },
		OnError:       func(err error) {},
	}
}

func (ws *BasicWebsocket) setConnected(connected bool) {
	ws.connMutex.Lock()
	ws.connected = connected
	ws.connMutex.Unlock()
}

// IsConnected is something
func (ws *BasicWebsocket) IsConnected() bool {
	ws.connMutex.Lock()
	defer ws.connMutex.Unlock()
	return ws.connected
}

// Connect to the server
func (ws *BasicWebsocket) Connect() error {
	ws.connMutex.Lock()
	defer ws.connMutex.Unlock()

	if ws.connected {
		return ErrAlreadyConnected
	}

	c, _, err := websocket.DefaultDialer.Dial(ws.url, ws.header)
	if err != nil {
		return err
	}

	ws.conn = c
	ws.connected = true

	go ws.startReader()
	go ws.startWriter()

	ws.OnConnect()

	return nil
}

func (ws *BasicWebsocket) startReader() {
	unexpectedStop := make(chan bool)

	go func() {
		for {
			messageType, message, err := ws.conn.ReadMessage()
			if err != nil {
				// conn.Close() was called, so just stop reading
				// see https://github.com/golang/go/issues/4373
				if strings.Contains(err.Error(), closedErrorMessage) {
					return
				}

				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
					// An unexpected error occurred, so notify the reader and reconnect if specified
					unexpectedStop <- true
				}
				return
			}

			if messageType == websocket.TextMessage {
				ws.readerMessages <- message
			}
		}
	}()

	for {
		select {
		case <-ws.readerDone:
			return
		case <-unexpectedStop:
			ws.Disconnect()
			return
		case message := <-ws.readerMessages:
			err := ws.OnMessage(message)
			if err != nil {
				ws.OnError(fmt.Errorf("handle message: %w", err))
			}
		}
	}
}

func (ws *BasicWebsocket) stopReader() {
	select {
	case ws.readerDone <- true:
	default:
		return
	}
}

func (ws *BasicWebsocket) startWriter() {
	for {
		select {
		case <-ws.writerDone:
			return
		case message := <-ws.writerMessages:
			err := ws.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				ws.OnError(fmt.Errorf("send message: %w", err))
			}
		}
	}
}

func (ws *BasicWebsocket) stopWriter() {
	ws.writerDone <- true

	// empty the channel
	for len(ws.writerMessages) > 0 {
		<-ws.writerMessages
	}
}

// ForceDisconnect sisconnect without reconnecting
func (ws *BasicWebsocket) ForceDisconnect() {
	ws.connMutex.Lock()
	ws.connected = false
	_ = ws.conn.Close()
	ws.stopReader()
	ws.stopWriter()
	ws.connMutex.Unlock()
}

// Disconnect and, if specified, reconnect afterwards. Returns whether reconnecting
func (ws *BasicWebsocket) Disconnect() bool {
	ws.ForceDisconnect()
	return ws.attemptReconnect()
}

// SendBytes send bytes to server (blocking)
func (ws *BasicWebsocket) SendBytes(bytes []byte) {
	ws.writerMessages <- bytes
}

// SendString send string to server (blocking)
func (ws *BasicWebsocket) SendString(s string) {
	ws.SendBytes([]byte(s))
}

// SendJSON marshal an interface into JSON, then send bytes to server (blocking)
func (ws *BasicWebsocket) SendJSON(data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	ws.SendBytes(bytes)
	return nil
}

func (ws *BasicWebsocket) attemptReconnect() bool {
	if ws.AutoReconnect {
		time.AfterFunc(ws.ReconnectTime, func() {
			_ = ws.Reconnect()
		})
		return true
	}
	return false
}

// Reconnect immediately disconnect and reconnect
func (ws *BasicWebsocket) Reconnect() error {
	if ws.connected {
		ws.ForceDisconnect()
	}

	err := ws.Connect()
	if err != nil {
		return err
	}

	return nil
}
