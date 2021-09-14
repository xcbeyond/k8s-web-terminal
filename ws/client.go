package ws

import (
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	EnableCompression: true,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WsMessage is WebSocket message
type WsMessage struct {
	MessageType int
	Data        []byte
}

// Client is WebSocket connection.
type Client struct {
	// a WebSocket connection.
	conn *websocket.Conn
	// read chan
	inChan chan *WsMessage
	// send chan
	outChan chan *WsMessage
	// mutex is avoid repeated closing pipe.
	mutex    sync.Mutex
	isClosed bool
	// close chan
	closeChan chan byte
}

// wsReadLoop is read goroutine.
func (c *Client) wsReadLoop() {
	var (
		msgType int
		data    []byte
		msg     *WsMessage
		err     error
	)
	for {
		// reading a message.
		if msgType, data, err = c.conn.ReadMessage(); err != nil {
			c.WsClose()
		}
		msg = &WsMessage{msgType, data}
		// putting in the request chan.
		select {
		case c.inChan <- msg:
		case <-c.closeChan:
			return
		}
	}
}

// wsWriteLoop is write goroutine.
func (c *Client) wsWriteLoop() {
	var msg *WsMessage
	var err error

	for {
		select {
		// get a reply.
		case msg = <-c.outChan:
			// write a message for WebSocket.
			if err = c.conn.WriteMessage(msg.MessageType, msg.Data); err != nil {
				c.WsClose()
				return
			}
		case <-c.closeChan:
			return
		}
	}
}

// InitWebsocket is init WebSocket connection.
func InitWebsocket(resp http.ResponseWriter, req *http.Request) (c *Client, err error) {
	var conn *websocket.Conn
	if conn, err = wsUpgrader.Upgrade(resp, req, nil); err != nil {
		return
	}
	c = &Client{
		conn:      conn,
		inChan:    make(chan *WsMessage, 1000),
		outChan:   make(chan *WsMessage, 1000),
		closeChan: make(chan byte),
		isClosed:  false,
	}

	go c.wsReadLoop()
	go c.wsWriteLoop()

	return
}

// WsWrite is send message.
func (c *Client) WsWrite(messageType int, data []byte) (err error) {
	select {
	case c.outChan <- &WsMessage{messageType, data}:
	case <-c.closeChan:
		err = errors.New("websocket closed")
	}
	return
}

// WsRead is read message.
func (c *Client) WsRead() (msg *WsMessage, err error) {
	select {
	case msg = <-c.inChan:
		return
	case <-c.closeChan:
		err = errors.New("websocket closed")
	}
	return
}

// WsClose is close WebSocket connection.
func (c *Client) WsClose() {
	c.conn.Close()

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if !c.isClosed {
		c.isClosed = true
		close(c.closeChan)
	}
}
