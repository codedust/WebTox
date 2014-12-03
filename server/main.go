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
	"flag"
	"fmt"
	"github.com/organ/golibtox"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
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

	// TODO load userstatus
	libtox.SetUserStatus(golibtox.USERSTATUS_NONE)

	toxid, err := libtox.GetAddress()
	if err != nil {
		panic(err)
	}
	fmt.Println("Tox ID:", strings.ToUpper(hex.EncodeToString(toxid)))

	// Register our callbacks
	libtox.CallbackFriendRequest(onFriendRequest)
	libtox.CallbackFriendMessage(onFriendMessage)
	libtox.CallbackConnectionStatus(onConnectionStatus)
	libtox.CallbackNameChange(onNameChange)
	libtox.CallbackStatusMessage(onStatusMessage)
	libtox.CallbackUserStatus(onUserStatus)
	libtox.CallbackFileSendRequest(onFileSendRequest)
	libtox.CallbackFileControl(onFileControl)
	libtox.CallbackFileData(onFileData)
	/**
	tox_callback_friend_action
	tox_callback_group_action
	tox_callback_group_invite
	tox_callback_group_message
	tox_callback_group_namelist_change
	tox_callback_read_receipt
	tox_callback_typing_change
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
