package main

import (
	"fmt"
	"github.com/codedust/go-tox"
	"golang.org/x/net/websocket"
	"strconv"
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

	awayOnDisconnectString, _ := storage.GetKeyValue("settings_away_on_disconnect")
	awayOnDisconnect, _ := strconv.ParseBool(awayOnDisconnectString)

	if awayOnDisconnect && len(activeConnections) == 1 {
		tox.SelfSetStatus(gotox.TOX_USERSTATUS_NONE)
		broadcastToClients(createSimpleJSONEvent("profile_update"))
	}

	for {
		if err = websocket.Message.Receive(conn, &clientMessage); err != nil {
			// the connection is closed
			fmt.Println("[handleWS] Read error. Removing client.", err.Error())
			delete(activeConnections, conn)
			fmt.Println("[handleWS] Number of clients still connected:", len(activeConnections))

			if len(activeConnections) == 0 {
				awayOnDisconnectString, _ := storage.GetKeyValue("settings_away_on_disconnect")
				awayOnDisconnect, _ := strconv.ParseBool(awayOnDisconnectString)

				if awayOnDisconnect {
					tox.SelfSetStatus(gotox.TOX_USERSTATUS_AWAY)
				}
			}
			return
		}
	}
})
