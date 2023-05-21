package main

import (
	"log"
	"time"

	"golang.org/x/net/websocket"
)

type client struct {
	socket   *websocket.Conn
	send     chan *message
	room     *room
	userData map[string]any
}

func (c *client) read() {
	defer c.socket.Close()
	for {
		var msg *message
		if err := websocket.JSON.Receive(c.socket, &msg); err == nil {
			msg.When = time.Now()
			msg.Name = c.userData["name"].(string)
			msg.AvatarURL, err = c.room.avatar.AvatarURL(c)
			if err != nil {
				log.Fatalln("AvatarURLの取得に失敗しました:", err)
			}
			c.room.forward <- msg
		} else {
			log.Printf("websocket.JSON.Receive error: %v", err)
			break
		}
	}
}

func (c *client) write() {
	defer c.socket.Close()
	for msg := range c.send {
		if err := websocket.JSON.Send(c.socket, msg); err != nil {
			log.Printf("websocket.JSON.Send error: %v", err)
			break
		}
	}
}
