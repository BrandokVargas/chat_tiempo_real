package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 512
	writeWait      = 10 * time.Second
	pinPeriod      = time.Minute
	pongWait       = pinPeriod + (10 * time.Second)
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Message struct {
	Nickname string `json:"nickname,omitempty"`
	Content  string `json:"content,omitempty"`
}

type Client struct {
	nickname     string
	hub          *Hub
	conn         *websocket.Conn
	queueMessage chan Message
}

func (c *Client) readWS() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	c.conn.SetPongHandler(func(ping string) error {
		fmt.Println("Pong: ", c.nickname, ping)
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		message := Message{}
		if err := c.conn.ReadJSON(&message); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("can't read the message ", err)
			}
			return
		}
		c.hub.broadcast <- message
	}
}

func (c *Client) writeWS() {

	ticker := time.NewTicker(pinPeriod)

	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, isOpen := <-c.queueMessage:

			if !isOpen {
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteJSON(message); err != nil {
				log.Println("can't write the message into ws: ", err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte("Ping")); err != nil {
				log.Println("can't write the message into ws: ", err)
				return
			}
		}
	}
}

func hanldeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	nickname := r.URL.Query()["nickname"]

	if len(nickname) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("cannot get the websocket connection: ", err)
		return
	}

	client := &Client{
		nickname:     nickname[0],
		hub:          hub,
		conn:         conn,
		queueMessage: make(chan Message, 2),
	}

	client.hub.register <- client
	go client.writeWS()
	go client.readWS()

}
