package main

import (
	"fmt"
	"golang.org/x/net/websocket"
)

type clientConn struct {
	websocket *websocket.Conn
	IP        string
}

var activeClients = make(map[clientConn]int)

func broadcastToClients(msg string) {
	go func() {
		for client, _ := range activeClients {
			if err := websocket.Message.Send(client.websocket, msg); err != nil {
				fmt.Println("[handleWS] Could not send message to ", client.IP, err.Error())
			}
		}
	}()
}

func handleWS(ws *websocket.Conn) {
	var err error
	var clientMessage string

	// cleanup on server side
	defer func() {
		if err = ws.Close(); err != nil {
			fmt.Println("[handleWS] Websocket could not be closed", err.Error())
		}
	}()

	clientIP := ws.Request().RemoteAddr
	fmt.Println("[handleWS] Client connected:", clientIP)

	client := clientConn{ws, clientIP}
	activeClients[client] = 0
	fmt.Println("[handleWS] Number of clients connected:", len(activeClients))

	for {
		if err = websocket.Message.Receive(ws, &clientMessage); err != nil {
			// the connection is closed
			fmt.Println("[handleWS] Read error. Removing client.", err.Error())
			delete(activeClients, client)
			fmt.Println("[handleWS] Number of clients still connected:", len(activeClients))
			return
		}
	}
}
