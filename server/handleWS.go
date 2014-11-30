package main

import (
	"fmt"
	"golang.org/x/net/websocket"
)

var activeConnections = make(map[*websocket.Conn]bool)

func broadcastToClients(msg string) {
	go func() {
		for conn, _ := range activeConnections {
			if err := websocket.Message.Send(conn, msg); err != nil {
				fmt.Println("[handleWS] Could not send message to ", conn.RemoteAddr, err.Error())
			}
		}
	}()
}

var handleWS = websocket.Handler(func(conn *websocket.Conn) {
	var err error
	var clientMessage string

	// cleanup on server side
	defer func() {
		if err = conn.Close(); err != nil {
			fmt.Println("[handleWS] Websocket could not be closed", err.Error())
		}
	}()

	activeConnections[conn] = true
	fmt.Println("[handleWS] Client connected:", conn.Request().RemoteAddr)
	fmt.Println("[handleWS] Number of clients connected:", len(activeConnections))

	for {
		if err = websocket.Message.Receive(conn, &clientMessage); err != nil {
			// the connection is closed
			fmt.Println("[handleWS] Read error. Removing client.", err.Error())
			delete(activeConnections, conn)
			fmt.Println("[handleWS] Number of clients still connected:", len(activeConnections))
			return
		}
	}
})
