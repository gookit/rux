// Package websocket utils
package websocket

import "github.com/gorilla/websocket"

// Refer links:
//
// gorilla/websocket https://github.com/gorilla/websocket
// golang.org/x/net/websocket
// golang.org/x/tools/playground/socket/socket.go

// WSApp struct
type WSApp struct {
	// all connections
	clients map[string]*websocket.Conn
}

// NewWSApp create new WSApp
func NewWSApp() *WSApp {
	return &WSApp{}
}
