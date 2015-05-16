package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/codedust/go-tox"
	"net/http"
	"strings"
)

var handleAPI = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")

	if libtox == nil {
		fmt.Println("[handleAPI] ERROR: libtox is nil.")
		rejectWithDefaultErrorJSON(w)
		return
	}

	request := r.URL.Path[len("/api"):]
	fmt.Println("[handleAPI]", request)

	switch {
	// GET REQUESTS
	case strings.HasPrefix(request, "/get/"):
		switch request {
		case "/get/contactlist":
			friendlist, err := getFriendListJSON()
			if err != nil {
				rejectWithDefaultErrorJSON(w)
			}
			fmt.Fprintf(w, friendlist)

		case "/get/profile":
			type profile struct {
				Username      string `json:"username"`
				StatusMessage string `json:"status_msg"`
				ToxID         string `json:"tox_id"`
				Status        string `json:"status"`
			}

			username, _ := libtox.SelfGetName()
			statusMessage, _ := libtox.SelfGetStatusMessage()
			toxid, _ := libtox.SelfGetAddress()
			status, _ := libtox.SelfGetStatus()
			p := profile{
				Username:      username,
				StatusMessage: string(statusMessage),
				ToxID:         strings.ToUpper(hex.EncodeToString(toxid)),
				Status:        getUserStatusAsString(status),
			}

			pJSON, _ := json.Marshal(p)
			fmt.Fprintf(w, string(pJSON))

		default:
			// unknown GET request
			rejectWithDefaultErrorJSON(w)
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

			_, err = libtox.FriendSendMessage(incomingData.Friend, gotox.MESSAGE_TYPE_NORMAL, incomingData.Message)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

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

			if err = libtox.SelfSetName(incomingData.Username); err != nil {
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

			if err = libtox.SelfSetStatus(getUserStatusFromString(incomingData.Status)); err != nil {
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

			if err = libtox.SelfSetStatusMessage(incomingData.StatusMessage); err != nil {
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
			if err != nil || len(friendAddressBytes) != gotox.ADDRESS_SIZE {
				rejectWithErrorJSON(w, "invalid_toxid", "The Tox ID you entered is invalid.")
				return
			}

			if len(incomingData.Message) == 0 {
				rejectWithErrorJSON(w, "no_message", "An invitation message is required.")
				return
			}

			friendID, err := libtox.FriendAdd(friendAddressBytes, incomingData.Message)
			if err != nil {
				switch err {
				case gotox.FaerrNoMessage:
					rejectWithErrorJSON(w, "no_message", "An invitation message is required.")
					return
				case gotox.FaerrTooLong:
					rejectWithErrorJSON(w, "invalid_message", "The message you entered is too long.")
					return
				case gotox.FaerrOwnKey:
					fallthrough
				case gotox.FaerrBadChecksum:
					fallthrough
				case gotox.FaerrSetNewNospam:
					rejectWithErrorJSON(w, "invalid_toxid", "The Tox ID you entered is invalid.")
					return
				case gotox.FaerrAlreadySent:
					rejectWithErrorJSON(w, "already_send", "A friend request to this person has already send.")
					return
				case gotox.FaerrUnkown:
					fallthrough
				case gotox.FaerrNoMem:
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

			err = libtox.FriendDelete(incomingData.Number)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

		default:
			// unknown POST request
			rejectWithDefaultErrorJSON(w)
		}

	default:
		// unknown API request
		rejectWithDefaultErrorJSON(w)
	}
})
