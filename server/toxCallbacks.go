package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/codedust/go-tox"
	"os"
	"time"
)

func onFriendRequest(t *gotox.Tox, publicKey []byte, message string) {
	fmt.Printf("New friend request from %s\n", hex.EncodeToString(publicKey))
	fmt.Printf("Invitation message: %v\n", message)

	// TODO Auto-accept friend request
	a, err := t.FriendAddNorequest(publicKey)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf(string(a))

	type jsonEvent struct {
		Type string `json:"type"`
	}

	e, _ := json.Marshal(jsonEvent{
		Type: "friendlist_update",
	})

	broadcastToClients(string(e))
}

func onFriendMessage(t *gotox.Tox, friendnumber uint32, messagetype gotox.ToxMessageType, message string) {
	fmt.Printf("New message from %d : %s\n", friendnumber, message)

	type jsonEvent struct {
		Type     string `json:"type"`
		Friend   uint32 `json:"friend"`
		Time     int64  `json:"time"`
		Message  string `json:"message"`
		IsAction bool   `json:"isAction"`
	}

	e, _ := json.Marshal(jsonEvent{
		Type:     "friend_message",
		Friend:   friendnumber,
		Time:     time.Now().Unix() * 1000,
		Message:  message,
		IsAction: messagetype == gotox.TOX_MESSAGE_TYPE_ACTION,
	})

	publicKey, _ := tox.FriendGetPublickey(friendnumber)
	storage.StoreMessage(hex.EncodeToString(publicKey), true, messagetype == gotox.TOX_MESSAGE_TYPE_ACTION, message)

	broadcastToClients(string(e))
}

func onFriendConnectionStatusChanges(t *gotox.Tox, friendnumber uint32, connectionStatus gotox.ToxConnection) {
	fmt.Printf("Connection status changed: %d -> %v\n", friendnumber, connectionStatus)

	type jsonEvent struct {
		Type   string `json:"type"`
		Friend uint32 `json:"friend"`
		Online bool   `json:"online"`
	}

	e, _ := json.Marshal(jsonEvent{
		Type:   "connection_status",
		Friend: friendnumber,
		Online: connectionStatus != gotox.TOX_CONNECTION_NONE,
	})

	broadcastToClients(string(e))
}

func onFriendNameChanges(t *gotox.Tox, friendnumber uint32, newname string) {
	fmt.Printf("Name changed: %d -> %s\n", friendnumber, newname)

	type jsonEvent struct {
		Type   string `json:"type"`
		Friend uint32 `json:"friend"`
		Name   string `json:"name"`
	}

	e, _ := json.Marshal(jsonEvent{
		Type:   "name_changed",
		Friend: friendnumber,
		Name:   newname,
	})

	broadcastToClients(string(e))
}

func onFriendStatusMessageChanges(t *gotox.Tox, friendnumber uint32, status string) {
	fmt.Printf("Status message changed: %d -> %s\n", friendnumber, status)

	type jsonEvent struct {
		Type      string `json:"type"`
		Friend    uint32 `json:"friend"`
		StatusMsg string `json:"status_msg"`
	}

	e, _ := json.Marshal(jsonEvent{
		Type:      "status_message_changed",
		Friend:    friendnumber,
		StatusMsg: status,
	})

	broadcastToClients(string(e))
}

func onFriendStatusChanges(t *gotox.Tox, friendnumber uint32, userstatus gotox.ToxUserStatus) {
	fmt.Printf("Friend status changed: %d -> %s\n", friendnumber, getUserStatusAsString(userstatus))

	type jsonEvent struct {
		Type   string `json:"type"`
		Friend uint32 `json:"friend"`
		Status string `json:"status"`
	}

	e, _ := json.Marshal(jsonEvent{
		Type:   "status_changed",
		Friend: friendnumber,
		Status: getUserStatusAsString(userstatus),
	})

	broadcastToClients(string(e))
}

func onFileRecv(t *gotox.Tox, friendnumber uint32, filenumber uint32, kind uint32, filesize uint64, filename string) {
	// Accept any file send request
	t.FileControl(friendnumber, true, filenumber, gotox.TOX_FILE_CONTROL_RESUME, nil)
	// Init *File handle
	f, _ := os.Create("example_" + filename)
	// Append f to the map[uint8]*os.File
	transfers[filenumber] = f
	transfersFilesizes[filenumber] = filesize
}

func onFileRecvControl(t *gotox.Tox, friendnumber uint32, filenumber uint32, fileControl gotox.ToxFileControl) {
	// TODO: Do something useful
}

func onFileRecvChunk(t *gotox.Tox, friendnumber uint32, filenumber uint32, position uint64, data []byte) {
	// Write data to the hopefully valid *File handle
	if f, exists := transfers[filenumber]; exists {
		f.WriteAt(data, (int64)(position))
	}

	// Finished receiving file
	if position == transfersFilesizes[filenumber] {
		f := transfers[filenumber]
		f.Sync()
		f.Close()
		delete(transfers, filenumber)
		fmt.Println("Written file", filenumber)
		t.FriendSendMessage(friendnumber, gotox.TOX_MESSAGE_TYPE_NORMAL, "Thanks!")
	}
}
