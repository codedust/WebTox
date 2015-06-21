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
	"./persistence"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/codedust/go-httpserve"
	"github.com/codedust/go-tox"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

// the global tox instance
var tox *gotox.Tox

// the global connection to the database
var storage *persistence.StorageConn

// Map of active file transfers
var transfers = make(map[uint32]*os.File)
var transfersFilesizes = make(map[uint32]uint64)

func main() {
	var newToxInstance bool = false
	var options *gotox.Options

	var err error
	storage, err = persistence.Open("../data/userdata.db")
	if err != nil {
		log.Panic("DB initialisation failed.")
	}
	defer storage.Close()

	//TODO change file location
	var toxSaveFilepath string
	flag.StringVar(&toxSaveFilepath, "p", filepath.Join(getUserprofilePath(), "webtox_save"), "path to save file")
	flag.Parse()
	fmt.Println("ToxData will be saved to", toxSaveFilepath)

	savedata, err := loadData(toxSaveFilepath)
	if err == nil {
		options = &gotox.Options{
			true, true,
			gotox.TOX_PROXY_TYPE_NONE, "127.0.0.1", 5555, 0, 0,
			3389,
			gotox.TOX_SAVEDATA_TYPE_TOX_SAVE, savedata}
	} else {
		options = &gotox.Options{
			true, true,
			gotox.TOX_PROXY_TYPE_NONE, "127.0.0.1", 5555, 0, 0,
			3389,
			gotox.TOX_SAVEDATA_TYPE_NONE, nil}
		newToxInstance = true
	}

	tox, err = gotox.New(options)
	if err != nil {
		panic(err)
	}

	var toxid []byte

	toxid, err = tox.SelfGetAddress()
	if err != nil {
		panic(err)
	}
	fmt.Println("Tox ID:", strings.ToUpper(hex.EncodeToString(toxid)))

	if newToxInstance {
		fmt.Println("Setting username to default: WebTox User")
		tox.SelfSetName("WebTox User")
		tox.SelfSetStatusMessage("WebToxing around...")
		tox.SelfSetStatus(gotox.TOX_USERSTATUS_NONE)
	} else {
		name, err := tox.SelfGetName()
		if err != nil {
			fmt.Println("Setting username to default: WebTox User")
			tox.SelfSetName("WebTox User")
		} else {
			fmt.Println("Username:", name)
		}

		if _, err = tox.SelfGetStatusMessage(); err != nil {
			if err = tox.SelfSetStatusMessage("WebToxing around..."); err != nil {
				panic(err)
			}
		}

		if _, err = tox.SelfGetStatus(); err != nil {
			if err = tox.SelfSetStatus(gotox.TOX_USERSTATUS_NONE); err != nil {
				panic(err)
			}
		}
	}

	// Register our callbacks
	tox.CallbackFriendRequest(onFriendRequest)
	tox.CallbackFriendMessage(onFriendMessage)
	tox.CallbackFriendConnectionStatusChanges(onFriendConnectionStatusChanges)
	tox.CallbackFriendNameChanges(onFriendNameChanges)
	tox.CallbackFriendStatusMessageChanges(onFriendStatusMessageChanges)
	tox.CallbackFriendStatusChanges(onFriendStatusChanges)
	tox.CallbackFileRecv(onFileRecv)
	tox.CallbackFileRecvControl(onFileRecvControl)
	tox.CallbackFileRecvChunk(onFileRecvChunk)

	// Connect to the network
	// TODO add more servers (as fallback)
	pubkey, _ := hex.DecodeString("04119E835DF3E78BACF0F84235B300546AF8B936F035185E2A8E9E0A67C8924F")
	err = tox.Bootstrap("144.76.60.215", 33445, pubkey)
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
			if err := saveData(tox, toxSaveFilepath); err != nil {
				fmt.Println(err)
			}

			fmt.Println("Killing")
			tox.Kill()
			return

		case <-ticker.C:
			tox.Iterate()
		}
	}
}

func serveGUI() {
	mux := http.NewServeMux()
	mux.Handle("/events", handleWS)
	mux.Handle("/api/", handleAPI)
	mux.Handle("/", http.FileServer(http.Dir("../html")))

	// add authentication
	salt, err := httpserve.RandomString(32)
	if err != nil {
		panic("could not generate salt")
	}

	// TODO store password in db and allow the user to change it
	handleAuth := httpserve.BasicAuthHandler(mux, "user", httpserve.Sha512Sum("pass"+salt), salt)

	// TODO support 0.0.0.0 and different ports
	httpserve.CreateCertificateIfNotExist(CFG_DATA_DIR+CFG_CERT_PREFIX+"cert.pem", CFG_DATA_DIR+CFG_CERT_PREFIX+"key.pem", "localhost", 3072)
	httpserve.ListenAndUpgradeTLS(":8080", CFG_DATA_DIR+CFG_CERT_PREFIX+"cert.pem", CFG_DATA_DIR+CFG_CERT_PREFIX+"key.pem", handleAuth)
}
