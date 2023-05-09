package main

import (
	"golang.org/x/net/websocket"
)

type client struct {
	socket *websocket.Conn
	send   chan []byte
	room   *room
}

func (c *client) read() {
	for {
		var msg []byte
		if err := websocket.Message.Receive(c.socket, &msg); err == nil {
			c.room.forward <- msg
		} else {
			break
		}
	}
	c.socket.Close()
}

func (c *client) write() {
	for msg := range c.send {
		if err := websocket.Message.Send(c.socket, string(msg)); err != nil {
			break
		}
	}
	c.socket.Close()
}
