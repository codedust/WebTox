package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/codedust/go-httpserve"
	"github.com/codedust/go-tox"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var handleAPI = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")

	if tox == nil {
		log.Print("[handleAPI] ERROR: tox is nil.")
		rejectWithDefaultErrorJSON(w)
		return
	}

	if storage == nil {
		log.Print("[handleAPI] ERROR: storage is nil.")
		rejectWithDefaultErrorJSON(w)
		return
	}

	request := r.URL.Path[len("/api"):]
	log.Println("[handleAPI]", request)

	switch {
	// GET REQUESTS
	case strings.HasPrefix(request, "/get/"):
		switch request {
		case "/get/contactlist":
			friendlist, err := getFriendListJSON()
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}
			fmt.Fprintf(w, friendlist)

		case "/get/profile":
			type profile struct {
				Username      string `json:"username"`
				StatusMessage string `json:"status_msg"`
				ToxID         string `json:"tox_id"`
				Status        string `json:"status"`
			}

			username, _ := tox.SelfGetName()
			statusMessage, _ := tox.SelfGetStatusMessage()
			toxid, _ := tox.SelfGetAddress()
			status, _ := tox.SelfGetStatus()
			p := profile{
				Username:      username,
				StatusMessage: string(statusMessage),
				ToxID:         strings.ToUpper(hex.EncodeToString(toxid)),
				Status:        getUserStatusAsString(status),
			}

			pJSON, _ := json.Marshal(p)
			fmt.Fprintf(w, string(pJSON))

		case "/get/settings":
			type settings struct {
				AuthUser string `json:"auth_user"`
				AwayOnDisconnect bool `json:"away_on_disconnect"`
				NotificationsEnabled bool `json:"notifications_enabled"`
			}

			username, _ := storage.GetKeyValue("settings_auth_user")
			notificationsEnabledString, _ := storage.GetKeyValue("settings_notifications_enabled")
			notificationsEnabled, _ := strconv.ParseBool(notificationsEnabledString)
			awayOnDisconnectString, _ := storage.GetKeyValue("settings_away_on_disconnect")
			awayOnDisconnect, _ := strconv.ParseBool(awayOnDisconnectString)

			s := settings{
				AuthUser: username,
				AwayOnDisconnect: awayOnDisconnect,
				NotificationsEnabled: notificationsEnabled,
			}

			sJSON, _ := json.Marshal(s)
			fmt.Fprintf(w, string(sJSON))

		default:
			// unknown GET request
			rejectWithDefaultErrorJSON(w)
			return
		}

	// POST REQUESTS
	case strings.HasPrefix(request, "/post/"):
		data := make([]byte, r.ContentLength)
		bytesRead, err := r.Body.Read(data)
		if err != nil && bytesRead == 0 {
			rejectWithDefaultErrorJSON(w)
			return
		}

		switch request {
		case "/post/message":
			type message struct {
				Friend  uint32
				Message string
			}

			var incomingData message
			err = json.Unmarshal(data, &incomingData)
			if err != nil || len(incomingData.Message) == 0 {
				rejectWithDefaultErrorJSON(w)
				return
			}

			_, err = tox.FriendSendMessage(incomingData.Friend, gotox.TOX_MESSAGE_TYPE_NORMAL, incomingData.Message)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

			publicKey, _ := tox.FriendGetPublickey(incomingData.Friend)
			storage.StoreMessage(hex.EncodeToString(publicKey), false, false, incomingData.Message)

			// broadcast message to all connected clients
			broadcastToClients(createSimpleJSONEvent("friendlist_update"))

		case "/post/message_read_receipt":
			type friend struct {
				Friend uint32 `json:"friend"`
			}

			var incomingData friend
			err = json.Unmarshal(data, &incomingData)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

			publicKey, _ := tox.FriendGetPublickey(incomingData.Friend)
			storage.SetLastMessageRead(hex.EncodeToString(publicKey))

		case "/post/username":
			type profile struct {
				Username string `json:"username"`
			}

			var incomingData profile
			err = json.Unmarshal(data, &incomingData)
			if err != nil || len(incomingData.Username) == 0 {
				rejectWithDefaultErrorJSON(w)
				return
			}

			if err = tox.SelfSetName(incomingData.Username); err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

		case "/post/status":
			type profile struct {
				Status string `json:"status"`
			}

			var incomingData profile
			err = json.Unmarshal(data, &incomingData)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

			if err = tox.SelfSetStatus(getUserStatusFromString(incomingData.Status)); err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

		case "/post/statusmessage":
			type profile struct {
				StatusMessage string `json:"status_msg"`
			}

			var incomingData profile
			err = json.Unmarshal(data, &incomingData)
			if err != nil || len(incomingData.StatusMessage) == 0 {
				rejectWithDefaultErrorJSON(w)
				return
			}

			if err = tox.SelfSetStatusMessage(incomingData.StatusMessage); err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

		case "/post/friend_request":
			type friendRequest struct {
				FriendID string `json:"friend_id"`
				Message  string `json:"message"`
			}

			var incomingData friendRequest
			err = json.Unmarshal(data, &incomingData)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

			friendAddressBytes, err := hex.DecodeString(incomingData.FriendID)
			if err != nil || len(friendAddressBytes) != gotox.TOX_ADDRESS_SIZE {
				rejectWithErrorJSON(w, "invalid_toxid", "The Tox ID you entered is invalid.")
				return
			}

			if len(incomingData.Message) == 0 {
				rejectWithErrorJSON(w, "no_message", "An invitation message is required.")
				return
			}

			friendID, err := tox.FriendAdd(friendAddressBytes, incomingData.Message)
			if err != nil {
				switch err {
				case gotox.ErrFriendAddNoMessage:
					rejectWithErrorJSON(w, "no_message", "An invitation message is required.")
					return
				case gotox.ErrFriendAddTooLong:
					rejectWithErrorJSON(w, "invalid_message", "The message you entered is too long.")
					return
				case gotox.ErrFriendAddOwnKey:
					fallthrough
				case gotox.ErrFriendAddBadChecksum:
					fallthrough
				case gotox.ErrFriendAddSetNewNospam:
					rejectWithErrorJSON(w, "invalid_toxid", "The Tox ID you entered is invalid.")
					return
				case gotox.ErrFriendAddAlreadySent:
					rejectWithErrorJSON(w, "already_send", "A friend request to this person has already send.")
					return
				case gotox.ErrFriendAddUnkown:
					fallthrough
				case gotox.ErrFriendAddNoMem:
				default:
					rejectWithDefaultErrorJSON(w)
					return
				}
			}
			fmt.Fprintf(w, string(friendID))

		case "/post/delete_friend":
			type friend struct {
				Number uint32 `json:"friend"`
			}

			var incomingData friend
			err = json.Unmarshal(data, &incomingData)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

			err = tox.FriendDelete(incomingData.Number)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

		case "/post/settings_auth_user":
			type user struct {
				Username string `json:"username"`
			}
			var incomingData user
			err = json.Unmarshal(data, &incomingData)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}
			err = storage.StoreKeyValue("settings_auth_user", incomingData.Username)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

			httpserve.ChangeAuthOptionsUser(authOptions, incomingData.Username)

		case "/post/settings_auth_pass":
			type user struct {
				Password string `json:"password"`
			}
			var incomingData user
			err = json.Unmarshal(data, &incomingData)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

			salt, err := httpserve.RandomString(32)
			if err != nil {
				panic("could not generate salt")
			}

			pass := httpserve.Sha512Sum(incomingData.Password + salt)

			err = storage.StoreKeyValue("settings_auth_pass", pass)
			err2 := storage.StoreKeyValue("settings_auth_salt", salt)

			if err != nil || err2 != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

			httpserve.ChangeAuthOptionsPass(authOptions, pass, salt)

		case "/post/keyValue":
			type keyValue struct {
				Key string `json:"key"`
				Value string `json:"value"`
			}

			var incomingData keyValue
			err = json.Unmarshal(data, &incomingData)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

			allowedKeys := map[string]bool {
				"settings_notifications_enabled": true,
				"settings_away_on_disconnect": true,
			}

			if !allowedKeys[incomingData.Key] {
				rejectWithDefaultErrorJSON(w)
				return
			}

			err = storage.StoreKeyValue(incomingData.Key, incomingData.Value)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

		default:
			// unknown POST request
			rejectWithDefaultErrorJSON(w)
			return
		}

	default:
		// unknown API request
		rejectWithDefaultErrorJSON(w)
		return
	}
})
