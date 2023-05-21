package main

import (
	"net/http"

	"github.com/dchf12/chat/trace"
	"github.com/stretchr/objx"
	"golang.org/x/net/websocket"
)

type room struct {
	forward chan *message
	join    chan *client
	leave   chan *client
	clients map[*client]bool
	tracer  trace.Tracer
	avatar  Avatar
}

func newRoom(avatar Avatar) *room {
	return &room{
		forward: make(chan *message),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
		avatar:  avatar,
	}
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			r.clients[client] = true
			r.tracer.Trace("新規クライアントが参加しました")
		case client := <-r.leave:
			delete(r.clients, client)
			close(client.send)
			r.tracer.Trace("クライアントが退出しました")
		case msg := <-r.forward:
			r.tracer.Trace("メッセージを受信しました: ", msg.Message)
			for client := range r.clients {
				select {
				case client.send <- msg:
					r.tracer.Trace(" -- クライアントに送信されました")
				default:
					delete(r.clients, client)
					close(client.send)
					r.tracer.Trace(" -- 送信に失敗しました。クライアントをクリーンアップします")
				}
			}
		}
	}
}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	websocket.Handler(func(ws *websocket.Conn) {
		authCookie, err := req.Cookie("auth")
		if err != nil {
			http.Error(w, "Cookieの取得に失敗しました", http.StatusForbidden)
			return
		}
		client := &client{
			socket:   ws,
			send:     make(chan *message, 256),
			room:     r,
			userData: objx.MustFromBase64(authCookie.Value),
		}
		r.join <- client
		defer func() { r.leave <- client }()
		go client.write()
		client.read()
	}).ServeHTTP(w, req)
}
