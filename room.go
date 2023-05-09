package main

import (
	"net/http"

	"golang.org/x/net/websocket"
)

type room struct {
	forward chan []byte
	join    chan *client
	leave   chan *client
	clients map[*client]bool
}

func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
	}
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			r.clients[client] = true
		case client := <-r.leave:
			delete(r.clients, client)
			close(client.send)
		case msg := <-r.forward:
			for client := range r.clients {
				select {
				case client.send <- msg:
				default:
					delete(r.clients, client)
					close(client.send)
				}
			}
		}
	}
}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	websocket.Handler(func(ws *websocket.Conn) {
		client := &client{
			socket: ws,
			send:   make(chan []byte, 256),
			room:   r,
		}
		r.join <- client
		defer func() { r.leave <- client }()
		go client.write()
		client.read()
	}).ServeHTTP(w, req)
}
