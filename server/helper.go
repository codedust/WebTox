package main

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/codedust/go-tox"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
)

func rejectWithErrorJSON(w http.ResponseWriter, code string, message string) {
	type Err struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	e := Err{Code: code, Message: message}
	jsonErr, _ := json.Marshal(e)
	http.Error(w, string(jsonErr), 422)
}

func rejectWithDefaultErrorJSON(w http.ResponseWriter) {
	type Err struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	e := Err{Code: "unknown", Message: "An unknown error occoured."}
	jsonErr, _ := json.Marshal(e)
	http.Error(w, string(jsonErr), 422)
}

func getUserStatusAsString(status gotox.UserStatus) string {
	switch status {
	case gotox.USERSTATUS_NONE:
		return "NONE"
	case gotox.USERSTATUS_AWAY:
		return "AWAY"
	case gotox.USERSTATUS_BUSY:
		return "BUSY"
	default:
		return "INVALID"
	}
}

func getUserStatusFromString(status string) gotox.UserStatus {
	switch status {
	case "NONE":
		return gotox.USERSTATUS_NONE
	case "AWAY":
		return gotox.USERSTATUS_AWAY
	case "BUSY":
		return gotox.USERSTATUS_BUSY
	default:
		return gotox.USERSTATUS_NONE
	}
}

func getFriendListJSON() (string, error) {
	type friend struct {
		Number    uint32   `json:"number"`
		ID        string   `json:"id"`
		Chat      []string `json:"chat"`
		Name      string   `json:"name"`
		Status    string   `json:"status"`
		StatusMsg string   `json:"status_msg"`
		Online    bool     `json:"online"`
	}

	friend_ids, err := libtox.SelfGetFriendlist()
	if err != nil {
		return "", err
	}

	friends := make([]friend, len(friend_ids))
	for i, friend_num := range friend_ids {
		// TODO: handle errors
		id, _ := libtox.FriendGetPublickey(friend_num)
		name, _ := libtox.FriendGetName(friend_num)
		connected, _ := libtox.FriendGetConnectionStatus(friend_num)
		userstatus, _ := libtox.FriendGetStatus(friend_num)
		status_msg, _ := libtox.FriendGetStatusMessage(friend_num)

		newfriend := friend{
			Number:    friend_num,
			ID:        strings.ToUpper(hex.EncodeToString(id)),
			Chat:      []string{},
			Name:      name,
			Status:    getUserStatusAsString(userstatus),
			StatusMsg: string(status_msg),
			Online:    connected != gotox.CONNECTION_NONE,
		}

		friends[i] = newfriend
	}
	jsonFriends, _ := json.Marshal(friends)
	return string(jsonFriends), nil
}

func getUserprofilePath() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func randomString(len int) string {
	bs := make([]byte, len)
	_, err := rand.Reader.Read(bs)
	if err != nil {
		// TODO
		panic("Error generating random string")
	}

	return base64.StdEncoding.EncodeToString(bs)
}

func sha512Sum(s string) string {
	hasher := sha512.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}

func loadData(filepath string) ([]byte, error) {
	if len(filepath) == 0 {
		return nil, errors.New("Empty path")
	}

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	return data, err
}

func saveData(t *gotox.Tox, filepath string) error {
	if len(filepath) == 0 {
		return errors.New("Empty path")
	}

	data, err := t.GetSavedata()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath, data, 0644)
	return err
}
