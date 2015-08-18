package main

import (
	"encoding/hex"
	"encoding/json"
	"github.com/codedust/go-tox"
	"log"
	"os"
	"time"
)

func onFriendRequest(t *gotox.Tox, publicKey []byte, message string) {
	log.Printf("New friend request from %s\n", hex.EncodeToString(publicKey))

	storage.StoreFriendRequest(hex.EncodeToString(publicKey), message)
	broadcastToClients(createSimpleJSONEvent("friend_requests_update"))
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
		publicKey, _ := tox.FriendGetPublickey(friendnumber)
		file, err := os.Create("../html/avatars/" + hex.EncodeToString(publicKey) + ".png")
		if err != nil {
			log.Println("[ERROR] Error creating file", "../html/avatars/"+hex.EncodeToString(publicKey)+".png")
		}

		// only accept avatars with a file size <= CFG_MAX_AVATAR_SIZE
		if filesize <= CFG_MAX_AVATAR_SIZE {
			// append the file to the map of active file transfers
			transfers[filenumber] = FileTransfer{fileHandle: file, fileSize: filesize, fileKind: kind}

			t.FileControl(friendnumber, filenumber, gotox.TOX_FILE_CONTROL_RESUME)
		} else {
			t.FileControl(friendnumber, filenumber, gotox.TOX_FILE_CONTROL_CANCEL)
		}

	} else if kind == gotox.TOX_FILE_KIND_DATA {
		file, err := os.Create("../html/download/" + filename)
		if err != nil {
			log.Println("[ERROR] Error creating file", "../html/download/"+filename)
		}

		// append the file to the map of active file transfers
		transfers[filenumber] = FileTransfer{fileHandle: file, fileSize: filesize, fileKind: kind}

		// TODO do not accept any file send request without asking the user
		t.FileControl(friendnumber, filenumber, gotox.TOX_FILE_CONTROL_RESUME)

	} else {
		log.Print("onFileRecv: unknown TOX_FILE_KIND: ", kind)
	}
}

func onFileRecvControl(t *gotox.Tox, friendnumber uint32, filenumber uint32, fileControl gotox.ToxFileControl) {
	transfer, ok := transfers[filenumber]
	if !ok {
		log.Println("Error: File handle does not exist")
		return
	}

	// TODO handle TOX_FILE_CONTROL_RESUME and TOX_FILE_CONTROL_PAUSE
	if fileControl == gotox.TOX_FILE_CONTROL_CANCEL {
		// delete file handle
		transfer.fileHandle.Close()
		delete(transfers, filenumber)
	}
}

func onFileRecvChunk(t *gotox.Tox, friendnumber uint32, filenumber uint32, position uint64, data []byte) {
	transfer, ok := transfers[filenumber]
	if !ok {
		if len(data) == 0 {
			// ignore the zero-length chunk that indicates that the transfer is
			// complete (see below)
			return
		}

		log.Println("Error: File handle does not exist")
		return
	}

	// write data to the file handle
	transfer.fileHandle.WriteAt(data, (int64)(position))

	// file transfer completed
	if position+uint64(len(data)) >= transfer.fileSize {
		// Some clients will send us another zero-length chunk without data (only
		// required for stream, not necessary for files with a known size) and some
		// will not.
		// We will delete the file handle now (we aleady reveived the whole file)
		// and ignore the file handle error when the empty chunk arrives.

		fileKind := transfer.fileKind

		transfer.fileHandle.Sync()
		transfer.fileHandle.Close()
		delete(transfers, filenumber)
		log.Println("File transfer completed (receiving)", filenumber)

		if fileKind == gotox.TOX_FILE_KIND_AVATAR {
			// update friendlist
			broadcastToClients(createSimpleJSONEvent("avatar_update"))
		}
	}
}
