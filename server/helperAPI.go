package main

import (
	"encoding/hex"
	"encoding/json"
	"github.com/codedust/go-tox"
)

// getFriendListJSON returns the users Tox friendlist as a JSON string
func getFriendListJSON() (string, error) {
	type Message struct {
		Message    string `json:"message"`
		IsIncoming bool   `json:"isIncoming"`
		IsAction   bool   `json:"isAction"`
		Time       int64  `json:"time"`
	}

	type friend struct {
		Number          uint32    `json:"number"`
		PublicKey       string    `json:"publicKey"`
		Chat            []Message `json:"chat"`
		LastMessageRead int64     `json:"last_msg_read"`
		Name            string    `json:"name"`
		Status          string    `json:"status"`
		StatusMsg       string    `json:"status_msg"`
		Online          bool      `json:"online"`
	}

	friend_ids, err := tox.SelfGetFriendlist()
	if err != nil {
		return "", err
	}

	friends := make([]friend, len(friend_ids))
	for i, friend_num := range friend_ids {
		// TODO: handle errors
		publicKey, _ := tox.FriendGetPublickey(friend_num)
		name, _ := tox.FriendGetName(friend_num)
		connected, _ := tox.FriendGetConnectionStatus(friend_num)
		userstatus, _ := tox.FriendGetStatus(friend_num)
		status_msg, _ := tox.FriendGetStatusMessage(friend_num)
		dbMessages := storage.GetMessages(hex.EncodeToString(publicKey), -1) // TOOD set a limit
		dbLastMessageRead, _ := storage.GetLastMessageRead(hex.EncodeToString(publicKey))

		var messages []Message

		for _, msg := range dbMessages {
			messages = append(messages, Message{Message: msg.Message, IsIncoming: msg.IsIncoming, IsAction: msg.IsAction, Time: msg.Time})
		}

		if messages == nil {
			messages = []Message{}
		}

		newfriend := friend{
			Number:          friend_num,
			PublicKey:       hex.EncodeToString(publicKey),
			Chat:            messages,
			LastMessageRead: dbLastMessageRead,
			Name:            name,
			Status:          getUserStatusAsString(userstatus),
			StatusMsg:       string(status_msg),
			Online:          connected != gotox.TOX_CONNECTION_NONE,
		}

		friends[i] = newfriend
	}
	jsonFriends, _ := json.Marshal(friends)
	return string(jsonFriends), nil
}
