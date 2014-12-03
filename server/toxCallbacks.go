package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/organ/golibtox"
	"os"
	"time"
)

func onFriendRequest(t *golibtox.Tox, publicKey []byte, data []byte, length uint16) {
	fmt.Printf("New friend request from %s\n", hex.EncodeToString(publicKey))
	fmt.Printf("Invitation message: %v\n", string(data))

	// TODO Auto-accept friend request
	a, err := t.AddFriendNorequest(publicKey)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf(string(a))
}

func onFriendMessage(t *golibtox.Tox, friendnumber int32, message []byte, length uint16) {
	fmt.Printf("New message from %d : %s\n", friendnumber, string(message))

	type jsonEvent struct {
		Type    string `json:"type"`
		Friend  int32  `json:"friend"`
		Time    int64  `json:"time"`
		Message string `json:"message"`
	}

	e, _ := json.Marshal(jsonEvent{
		Type:    "friend_message",
		Friend:  friendnumber,
		Time:    time.Now().Unix() * 1000,
		Message: string(message),
	})

	// TODO save messages server-side

	broadcastToClients(string(e))
}

func onConnectionStatus(t *golibtox.Tox, friendnumber int32, online bool) {
	fmt.Printf("Status changed: %d -> %t\n", friendnumber, online)

	type jsonEvent struct {
		Type   string `json:"type"`
		Friend int32  `json:"friend"`
		Online bool   `json:"online"`
	}

	e, _ := json.Marshal(jsonEvent{
		Type:   "connection_status",
		Friend: friendnumber,
		Online: online,
	})

	broadcastToClients(string(e))
}

func onNameChange(t *golibtox.Tox, friendnumber int32, newname []byte, length uint16) {
	fmt.Printf("Name changed: %d -> %s\n", friendnumber, newname)

	type jsonEvent struct {
		Type   string `json:"type"`
		Friend int32  `json:"friend"`
		Name   string `json:"name"`
	}

	e, _ := json.Marshal(jsonEvent{
		Type:   "name_changed",
		Friend: friendnumber,
		Name:   string(newname),
	})

	broadcastToClients(string(e))
}

func onStatusMessage(t *golibtox.Tox, friendnumber int32, status []byte, length uint16) {
	fmt.Printf("Status message changed: %d -> %s\n", friendnumber, status)

	type jsonEvent struct {
		Type      string `json:"type"`
		Friend    int32  `json:"friend"`
		StatusMsg string `json:"status_msg"`
	}

	e, _ := json.Marshal(jsonEvent{
		Type:      "status_message_changed",
		Friend:    friendnumber,
		StatusMsg: string(status),
	})

	broadcastToClients(string(e))
}

func onUserStatus(t *golibtox.Tox, friendnumber int32, userstatus golibtox.UserStatus) {
	fmt.Printf("Status changed: %d -> %s\n", friendnumber, userstatusString(userstatus))

	type jsonEvent struct {
		Type   string `json:"type"`
		Friend int32  `json:"friend"`
		Status string `json:"status"`
	}

	e, _ := json.Marshal(jsonEvent{
		Type:   "status_changed",
		Friend: friendnumber,
		Status: userstatusString(userstatus),
	})

	broadcastToClients(string(e))
}

func onFileSendRequest(t *golibtox.Tox, friendnumber int32, filenumber uint8, filesize uint64, filename []byte, filenameLength uint16) {
	// TODO
	// Accept any file send request
	t.FileSendControl(friendnumber, true, filenumber, golibtox.FILECONTROL_ACCEPT, nil)
	// Init *File handle
	f, _ := os.Create("example_" + string(filename))
	// Append f to the map[uint8]*os.File
	transfers[filenumber] = f
}

func onFileControl(t *golibtox.Tox, friendnumber int32, sending bool, filenumber uint8, fileControl golibtox.FileControl, data []byte, length uint16) {
	// TODO
	// Finished receiving file
	if fileControl == golibtox.FILECONTROL_FINISHED {
		f := transfers[filenumber]
		f.Sync()
		f.Close()
		delete(transfers, filenumber)
		fmt.Println("Written file", filenumber)
		t.SendMessage(friendnumber, []byte("Finished! Thanks!"))
	}
}

func onFileData(t *golibtox.Tox, friendnumber int32, filenumber uint8, data []byte, length uint16) {
	// TODO
	// Write data to the hopefully valid *File handle
	if f, exists := transfers[filenumber]; exists {
		f.Write(data)
	}
}
