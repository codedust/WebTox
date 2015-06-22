package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
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

	broadcastToClients(createSimpleJSONEvent("friendlist_update"))
}

func onFriendMessage(t *gotox.Tox, friendnumber uint32, messagetype gotox.ToxMessageType, message string) {
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

func onFileRecv(t *gotox.Tox, friendnumber uint32, filenumber uint32, kind gotox.ToxFileKind, filesize uint64, filename string) {
	if kind == gotox.TOX_FILE_KIND_AVATAR {
		// Init *File handle
		publicKey, _ := tox.FriendGetPublickey(friendnumber)
		f, _ := os.Create("../html/avatars/" + hex.EncodeToString(publicKey) + ".png")

		// Append f to the map[uint8]*os.File
		transfers[filenumber] = f
		transfersFilesizes[filenumber] = filesize
		t.FileControl(friendnumber, true, filenumber, gotox.TOX_FILE_CONTROL_RESUME, nil)

	} else if kind == gotox.TOX_FILE_KIND_DATA {
		// Init *File handle
		f, _ := os.Create("../html/download" + filename)

		// Append f to the map[uint8]*os.File
		transfers[filenumber] = f
		transfersFilesizes[filenumber] = filesize

		// TODO do not accept any file send request without asking the user
		t.FileControl(friendnumber, true, filenumber, gotox.TOX_FILE_CONTROL_RESUME, nil)

	} else {
		log.Print("onFileRecv: unknown TOX_FILE_KIND: ", kind)
	}
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
		log.Println("File written: ", filenumber)
		t.FriendSendMessage(friendnumber, gotox.TOX_MESSAGE_TYPE_ACTION, "File transfer completed.")

		// update friendlist (avatar updates)
		broadcastToClients(createSimpleJSONEvent("avatar_update"))
	}
}
