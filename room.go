package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/dchf12/chat/trace"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return isAllowedWebSocketOrigin(r)
	},
}

type room struct {
	forward chan *message
	join    chan *client
	leave   chan *client
	clients map[*client]struct{}
	tracer  trace.Tracer
	avatar  Avatar
	done    chan struct{}
}

func newRoom(avatar Avatar) *room {
	return &room{
		forward: make(chan *message),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]struct{}),
		avatar:  avatar,
		done:    make(chan struct{}),
	}
}

func (r *room) run() {
	for {
		select {
		case <-r.done:
			return
		case client := <-r.join:
			r.clients[client] = struct{}{}
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

func (r *room) Stop() {
	close(r.done)
}

func (r *room) WebSocketHandler(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	userData, err := getAuthUserData(c)
	if err != nil {
		_ = ws.Close()
		return c.String(http.StatusForbidden, "Cookieの取得に失敗しました")
	}

	client := &client{
		socket:   ws,
		send:     make(chan *message, 256),
		room:     r,
		userData: userData,
	}
	r.join <- client
	defer func() { r.leave <- client }()
	go client.write()
	client.read()

	return nil
}

func isAllowedWebSocketOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}

	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}

	if !strings.EqualFold(u.Host, r.Host) {
		return false
	}

	if r.TLS != nil {
		return strings.EqualFold(u.Scheme, "https")
	}
	return strings.EqualFold(u.Scheme, "http")
}
