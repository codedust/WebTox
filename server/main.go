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
	"github.com/codedust/go-tox"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

type Server struct {
	Address   string
	Port      uint16
	PublicKey []byte
}

var libtox *gotox.Tox

// Map of active file transfers
var transfers = make(map[uint32]*os.File)
var transfersFilesizes = make(map[uint32]uint64)

func main() {
	var newToxInstance bool = false

	//TODO change file location
	var toxSaveFilepath string
	flag.StringVar(&toxSaveFilepath, "p", filepath.Join(getUserprofilePath(), "webtox_save"), "path to save file")
	flag.Parse()
	fmt.Println("Data will be saved to", toxSaveFilepath)

	data, err := loadData(toxSaveFilepath)
	if err != nil {
		newToxInstance = true
	}

	o := &gotox.Options{true, true, gotox.PROXY_TYPE_NONE, "127.0.0.1", 5555, 0, 0}
	libtox, err = gotox.New(o, data)
	if err != nil {
		panic(err)
	}

	var toxid []byte

	toxid, err = libtox.SelfGetAddress()
	if err != nil {
		panic(err)
	}
	fmt.Println("Tox ID:", strings.ToUpper(hex.EncodeToString(toxid)))

	if newToxInstance {
		fmt.Println("Setting username to default: WebTox User")
		libtox.SelfSetName("WebTox User")
		libtox.SelfSetStatusMessage("WebToxing around...")
		libtox.SelfSetStatus(gotox.USERSTATUS_NONE)
	} else {
		name, err := libtox.SelfGetName()
		if err != nil {
			fmt.Println("Setting username to default: WebTox User")
			libtox.SelfSetName("WebTox User")
		} else {
			fmt.Println("Username:", name)
		}

		if _, err = libtox.SelfGetStatusMessage(); err != nil {
			if err = libtox.SelfSetStatusMessage("WebToxing around..."); err != nil {
				panic(err)
			}
		}

		if _, err = libtox.SelfGetStatus(); err != nil {
			if err = libtox.SelfSetStatus(gotox.USERSTATUS_NONE); err != nil {
				panic(err)
			}
		}
	}

	// Register our callbacks
	libtox.CallbackFriendRequest(onFriendRequest)
	libtox.CallbackFriendMessage(onFriendMessage)
	libtox.CallbackFriendConnectionStatusChanges(onFriendConnectionStatusChanges)
	libtox.CallbackFriendNameChanges(onFriendNameChanges)
	libtox.CallbackFriendStatusMessageChanges(onFriendStatusMessageChanges)
	libtox.CallbackFriendStatusChanges(onFriendStatusChanges)
	libtox.CallbackFileRecv(onFileRecv)
	libtox.CallbackFileRecvControl(onFileRecvControl)
	libtox.CallbackFileRecvChunk(onFileRecvChunk)

	// Connect to the network
	// TODO add more servers (as fallback)
	pubkey, _ := hex.DecodeString("04119E835DF3E78BACF0F84235B300546AF8B936F035185E2A8E9E0A67C8924F")
	server := &Server{"144.76.60.215", 33445, pubkey}

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
			libtox.Iterate()
		}
	}
}
