package main

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type client struct {
	socket   *websocket.Conn
	send     chan *message
	room     *room
	userData map[string]any
}

func (c *client) read() {
	defer func() { _ = c.socket.Close() }()
	for {
		var msg *message
		if err := c.socket.ReadJSON(&msg); err == nil {
			msg.When = time.Now()
			name, ok := c.userData["name"].(string)
			if !ok {
				log.Printf("invalid userData: name is missing or not a string")
				break
			}
			msg.Name = name

			var err error
			msg.AvatarURL, err = c.room.avatar.AvatarURL(c)
			if err != nil {
				log.Printf("failed to get avatar URL: %v", err)
				continue
			}
			c.room.forward <- msg
		} else {
			log.Printf("websocket read error: %v", err)
			break
		}
	}
}

func (c *client) write() {
	defer func() { _ = c.socket.Close() }()
	for msg := range c.send {
		if err := c.socket.WriteJSON(msg); err != nil {
			log.Printf("websocket write error: %v", err)
			break
		}
	}
}
