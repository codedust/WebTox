package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/organ/golibtox"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func serveGUI() {
	go ws_hub.run()
	http.HandleFunc("/events", handleWS)
	http.HandleFunc("/api/", handleAPI)
	http.HandleFunc("/", handleHTTP)

	// TODO support 0.0.0.0 and different ports
	//err := http.ListenAndServe("127.0.0.1:8080", nil)
	err := http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		panic(err)
	}
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
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
				// TODO: status
			}

			username, _ := libtox.GetSelfName()
			statusMessage, _ := libtox.GetSelfStatusMessage()
			toxid, _ := libtox.GetAddress()
			p := profile{
				Username:      username,
				StatusMessage: string(statusMessage),
				ToxID:         strings.ToUpper(hex.EncodeToString(toxid)),
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
				Friend  int32
				Message string
			}

			var incomingData message
			err = json.Unmarshal(data, &incomingData)
			if err != nil || len(incomingData.Message) == 0 {
				rejectWithDefaultErrorJSON(w)
				return
			}

			_, err = libtox.SendMessage(incomingData.Friend, []byte(incomingData.Message))
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

			if err = libtox.SetName(incomingData.Username); err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

		case "/post/status":
			// TODO

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

			if err = libtox.SetStatusMessage([]byte(incomingData.StatusMessage)); err != nil {
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

			FriendIDbyte, err := hex.DecodeString(incomingData.FriendID)
			if err != nil || len(FriendIDbyte) != golibtox.FRIEND_ADDRESS_SIZE {
				rejectWithErrorJSON(w, "invalid_toxid", "The Tox ID you entered is invalid.")
				return
			}

			if len(incomingData.Message) == 0 {
				rejectWithErrorJSON(w, "no_message", "An invitation message is required.")
				return
			}

			friendID, err := libtox.AddFriend(FriendIDbyte, []byte(incomingData.Message))
			if err != nil {
				switch err {
				case golibtox.FaerrNoMessage:
					rejectWithErrorJSON(w, "no_message", "An invitation message is required.")
					return

				case golibtox.FaerrTooLong:
					rejectWithErrorJSON(w, "invalid_message", "The message you entered is too long.")
					return

				case golibtox.FaerrOwnKey:
					fallthrough
				case golibtox.FaerrBadChecksum:
					fallthrough
				case golibtox.FaerrSetNewNospam:
					rejectWithErrorJSON(w, "invalid_toxid", "The Tox ID you entered is invalid.")
					return

				case golibtox.FaerrAlreadySent:
					rejectWithErrorJSON(w, "already_send", "A friend request to this person has already send.")
					return

				case golibtox.FaerrUnkown:
					fallthrough
				case golibtox.FaerrNoMem:
				default:
					rejectWithDefaultErrorJSON(w)
					return
				}
			}
			fmt.Fprintf(w, string(friendID))

		case "/post/delete_friend":
			type friend struct {
				Number int32 `json:"friend"`
			}

			var incomingData friend
			err = json.Unmarshal(data, &incomingData)
			if err != nil {
				rejectWithDefaultErrorJSON(w)
				return
			}

			err = libtox.DelFriend(incomingData.Number)
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
}

func handleHTTP(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Path
	fmt.Println("[handleHTTP]", file)

	if file[0] == '/' {
		file = file[1:]
	}

	if len(file) == 0 {
		file = "index.html"
	}

	p := filepath.Join("../html", filepath.FromSlash(file))
	_, err := os.Stat(p)
	if err == nil {
		http.ServeFile(w, r, p)
		return
	} else {
		http.NotFound(w, r)
		return
	}
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[handleWS]", r.URL.Path)

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	c := &connection{send: make(chan []byte, 256), ws: ws}
	ws_hub.register <- c

	go c.writePump()
	c.readPump()
}
