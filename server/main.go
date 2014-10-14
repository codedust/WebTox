/*
	WebTox - A web based graphical user interface for Tox
	Copyright (C) 2014 WebTox authors and contributers

	This file is part of WebTox.

	WebTox is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	WebTox is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with WebTox.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/organ/golibtox"
)

type Server struct {
	Address   string
	Port      uint16
	PublicKey string
}

var libtox *golibtox.Tox

// TODO
// Map of active file transfers
var transfers = make(map[uint8]*os.File)

func main() {
	var err error
	libtox, err = golibtox.New(&golibtox.Options{true, false, false, "127.0.0.1", 5555})
	if err != nil {
		panic(err)
	}

	//TODO change file location
	var toxSaveFilepath string
	flag.StringVar(&toxSaveFilepath, "p", filepath.Join(getUserprofilePath(), "webtox_save"), "path to save file")
	flag.Parse()
	fmt.Println("Data will be saved to", toxSaveFilepath)

	if err := loadData(libtox, toxSaveFilepath); err != nil {
		fmt.Println("Setting username to default: WebTox User")
		libtox.SetName("WebTox User")
		libtox.SetStatusMessage([]byte("WebToxing around..."))
	} else {
		name, err := libtox.GetSelfName()
		if err != nil {
			fmt.Println("Setting username to default: WebTox User")
			libtox.SetName("WebTox User")
		}
		fmt.Println("Username:", name)
	}

	// TODO
	libtox.SetUserStatus(golibtox.USERSTATUS_NONE)

	toxid, err := libtox.GetAddress()
	if err != nil {
		panic(err)
	}
	fmt.Println("Tox ID:", strings.ToUpper(hex.EncodeToString(toxid)))

	// Register our callbacks
	libtox.CallbackFriendRequest(onFriendRequest)
	libtox.CallbackFriendMessage(onFriendMessage)
	libtox.CallbackFileSendRequest(onFileSendRequest)
	libtox.CallbackFileControl(onFileControl)
	libtox.CallbackFileData(onFileData)
	libtox.CallbackConnectionStatus(onConnectionStatus)
	/**
	tox_callback_friend_action
	tox_callback_group_action
	tox_callback_group_invite
	tox_callback_group_message
	tox_callback_group_namelist_change
	tox_callback_name_change
	tox_callback_read_receipt
	tox_callback_status_message
	tox_callback_typing_change
	tox_callback_user_status
	**/

	// Connect to the network
	// TODO add more servers (as fallback)
	server := &Server{"192.254.75.98", 33445, "951C88B7E75C867418ACDB5D273821372BB5BD652740BCDF623A4FA293E75D2F"}

	err = libtox.BootstrapFromAddress(server.Address, server.Port, server.PublicKey)
	if err != nil {
		panic(err)
	}

	// Start the server
	go serveGUI()

	// Main loop
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	ticker := time.NewTicker(25 * time.Millisecond)

	for {
		select {
		case <-c:
			fmt.Println("Saving...")
			if err := saveData(libtox, toxSaveFilepath); err != nil {
				fmt.Println(err)
			}

			fmt.Println("Killing")
			libtox.Kill()
			return

		case <-ticker.C:
			libtox.Do()
		}
	}
}

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
	// broadcast message to ws clients
	ws_hub.broadcast <- []byte(e)
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

	ws_hub.broadcast <- []byte(e)
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

func loadData(t *golibtox.Tox, filepath string) error {
	if len(filepath) == 0 {
		return errors.New("Empty path")
	}

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	err = t.Load(data)
	return err
}

func saveData(t *golibtox.Tox, filepath string) error {
	if len(filepath) == 0 {
		return errors.New("Empty path")
	}

	data, err := t.Save()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath, data, 0644)
	return err
}
