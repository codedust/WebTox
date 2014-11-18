package main

import (
	"encoding/hex"
	"encoding/json"
	"github.com/organ/golibtox"
	"net/http"
	"os"
	"runtime"
	"strings"
	"errors"
	"io/ioutil"
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

func userstatusString(status golibtox.UserStatus) string {
	switch status {
	case golibtox.USERSTATUS_NONE:
		return "NONE"
	case golibtox.USERSTATUS_AWAY:
		return "AWAY"
	case golibtox.USERSTATUS_BUSY:
		return "BUSY"
	default:
		return "INVALID"
	}
}

func getFriendListJSON() (string, error) {
	type friend struct {
		Number    int32    `json:"number"`
		ID        string   `json:"id"`
		Chat      []string `json:"chat"`
		Name      string   `json:"name"`
		Status    string   `json:"status"`
		StatusMsg string   `json:"status_msg"`
		Online    bool     `json:"online"`
	}

	friend_ids, err := libtox.GetFriendlist()
	if err != nil {
		return "", err
	}

	friends := make([]friend, len(friend_ids))
	for i, friend_num := range friend_ids {
		// TODO: handle errors
		id, _ := libtox.GetClientId(friend_num)
		name, _ := libtox.GetName(friend_num)
		connected, _ := libtox.GetFriendConnectionStatus(friend_num)
		userstatus, _ := libtox.GetUserStatus(friend_num)
		status_msg, _ := libtox.GetStatusMessage(friend_num)

		newfriend := friend{
			Number:    friend_num,
			ID:        strings.ToUpper(hex.EncodeToString(id)),
			Chat:      []string{},
			Name:      name,
			Status:    userstatusString(userstatus),
			StatusMsg: string(status_msg),
			Online:    connected,
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
