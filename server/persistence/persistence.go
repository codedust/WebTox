package persistence

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"sync"
	"time"
)

var (
	KeyNotFound = errors.New("Key does not exist")
)

type StorageConn struct {
	db  *sql.DB
	mtx sync.Mutex
}

type Message struct {
	Message    string
	IsIncoming bool
	IsAction   bool
	Time       int64
}

type FriendRequest struct {
	PublicKey string
	Message   string
	IsIgnored bool
}

// Open creates a connection to the database
// always close the connection with `defer storageConn.Close()`
func Open(filename string) (*StorageConn, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		log.Fatal(err)
		return &StorageConn{}, err
	}

	// create database tables
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY,
		friend INTEGER,
		isIncoming INTEGER,
		isAction INTEGER,
		time INTEGER,
		message TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS friends (
		id INTEGER PRIMARY KEY,
		publicKey TEXT
	);
	CREATE TABLE IF NOT EXISTS friend_requests (
		publicKey TEXT NOT NULL,
		message TEXT NOT NULL,
		isIgnored INTEGER,
		PRIMARY KEY(publicKey)
	);
	CREATE TABLE IF NOT EXISTS friendLastMessageRead (
		friend INTEGER PRIMARY KEY,
		time INTEGER
	);
	CREATE TABLE IF NOT EXISTS keyValueStorage (
		key TEXT PRIMARY KEY,
		value TEXT
	);`

	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Panicf("%q: %s\n", err, sqlStmt)
		return &StorageConn{}, err
	}

	s := &StorageConn{db: db}
	return s, nil
}

// Close safely closes the connection to the database
func (s *StorageConn) Close() {
	s.db.Close()
}

// StoreKeyValue stores a key-value pair of strings
// key    the key (case sensitive)
// value  the value
func (s *StorageConn) StoreKeyValue(key string, value string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, err := s.db.Exec(`INSERT OR REPLACE INTO keyValueStorage(key, value) VALUES(?, ?)`, key, value)
	if err != nil {
		log.Print("[persistence StoreKeyValue] INSERT statement failed")
		return err
	}

	return nil
}

// GetKeyValue returns the value corresponding to the given key or an empty
// string if the value could not be determined
// key  the key (case sensitive)
func (s *StorageConn) GetKeyValue(key string) (string, error) {
	rows, err := s.db.Query("SELECT value FROM keyValueStorage WHERE key = ?", key)
	if err != nil {
		log.Print("[persistence GetKeyValue] SELECT statement failed")
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var value string
		rows.Scan(&value)
		return value, nil
	}

	return "", KeyNotFound
}

// StoreMessage stores a message
// friendPublicKey  the publicKey of the friend
// isIncoming       specifies if the message is received (true) or sent (false)
// isAction         specifies if the message is an action or not
// message          the message
func (s *StorageConn) StoreMessage(friendPublicKey string, isIncoming bool, isAction bool, message string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	friendID, err := s.getFriendDbId(friendPublicKey)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`INSERT INTO messages(friend, isIncoming, isAction, time, message) VALUES(?, ?, ?, ?, ?)`, friendID, isIncoming, isAction, time.Now().Unix()*1000, message)
	if err != nil {
		log.Print("[persistence StoreMessage] INSERT statement failed")
		return err
	}
	return nil
}

// GetMessages returns previously stored messages of a friend.
// friendPublicKey  the publicKey of the friend
// limit            the number of messages that should be returned. Set limit
//                  to -1 to get all messages
func (s *StorageConn) GetMessages(friendPublicKey string, limit int) []Message {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	friendId, err := s.getFriendDbId(friendPublicKey)
	if err != nil {
		log.Print("[persistence GetMessages] getFriendDbId failed")
		return nil
	}

	rows, err := s.db.Query("SELECT isAction, isIncoming, time, message FROM messages WHERE friend = ? ORDER BY id DESC LIMIT ?", friendId, limit)
	if err != nil {
		log.Print("[persistence GetMessages] SELECT statement failed")
		return nil
	}
	defer rows.Close()

	var messages []Message

	for rows.Next() {
		var isIncoming bool
		var isAction bool
		var time int64
		var message string
		rows.Scan(&isAction, &isIncoming, &time, &message)
		messages = append(messages, Message{Message: message, IsIncoming: isIncoming, IsAction: isAction, Time: time})
	}

	if messages == nil {
		return nil
	}

	return messages
}

// StoreFriendRequest stores a friend request
// friendPublicKey  the publicKey of the friend request
// message          the message send with the friend request
func (s *StorageConn) StoreFriendRequest(friendPublicKey string, message string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, err := s.db.Exec(`INSERT OR REPLACE INTO friend_requests(publicKey, message, isIgnored) VALUES(?, ?, ?)`, friendPublicKey, message, 0)
	if err != nil {
		log.Print("[persistence StoreMessage] INSERT statement failed")
		return err
	}
	return nil
}

// GetFriendRequests returns previously stored friend requests.
// limit  the number of friend requests that should be returned. Set limit to
//        -1 to get all messages
func (s *StorageConn) GetFriendRequests(limit int) []FriendRequest {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	rows, err := s.db.Query("SELECT publicKey, message, isIgnored FROM friend_requests ORDER BY isIgnored DESC LIMIT ?", limit)
	if err != nil {
		log.Print("[persistence GetMessages] SELECT statement failed")
		return nil
	}
	defer rows.Close()

	var friendRequests []FriendRequest

	for rows.Next() {
		var publicKey string
		var message string
		var isIgnored bool
		rows.Scan(&publicKey, &message, &isIgnored)
		friendRequests = append(friendRequests, FriendRequest{PublicKey: publicKey, Message: message, IsIgnored: isIgnored})
	}

	if friendRequests == nil {
		return nil
	}

	return friendRequests
}

// StoreFriendRequestIgnoreStatus updates the isIgnored attribute of a friend request
// friendPublicKey  the publicKey of the friend request
// isIgnored        the new value for the isIgnored attribute
func (s *StorageConn) StoreFriendRequestIgnoreStatus(friendPublicKey string, isIgnored bool) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, err := s.db.Exec(`UPDATE friend_requests SET isIgnored = ? WHERE publicKey = ?`, isIgnored, friendPublicKey)
	if err != nil {
		log.Print("[persistence StoreMessage] UPDATE statement failed")
		return err
	}
	return nil
}

// DeleteFriendRequest deletes a stored friend request
// friendPublicKey  the publicKey of the friend request
func (s *StorageConn) DeleteFriendRequest(friendPublicKey string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, err := s.db.Exec(`DELETE FROM friend_requests WHERE publicKey = ?`, friendPublicKey)
	if err != nil {
		log.Print("[persistence StoreMessage] DELETE statement failed")
		return err
	}
	return nil
}

// SetLastMessageRead sets the message read status for a given friend
// friendPublicKey  the publicKey of the friend
func (s *StorageConn) SetLastMessageRead(friendPublicKey string) error {
	friendId, err := s.getFriendDbId(friendPublicKey)
	if err != nil {
		log.Print("[persistence GetMessages] getFriendDbId failed")
		return err
	}

	_, err = s.db.Exec(`INSERT OR REPLACE INTO friendLastMessageRead(friend, time) VALUES(?, ?)`, friendId, time.Now().Unix()*1000)
	if err != nil {
		log.Print("[persistence setLastMessageRead] INSERT statement failed")
		return err
	}

	return nil
}

// GetLastMessageRead returns the last message read time for a given friend
// friendPublicKey  the publicKey of the friend
func (s *StorageConn) GetLastMessageRead(friendPublicKey string) (int64, error) {
	friendId, err := s.getFriendDbId(friendPublicKey)
	if err != nil {
		log.Print("[persistence GetMessages] getFriendDbId failed")
		return 0, err
	}

	rows, err := s.db.Query("SELECT time FROM friendLastMessageRead WHERE friend = ?", friendId)
	if err != nil {
		log.Print("[persistence getLastMessageRead] SELECT statement failed")
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		var time int64
		rows.Scan(&time)
		return time, nil
	}

	return 0, nil
}

// getFriendDbId returns the friendId that is used internally in the database
// for the friend with the given publicKey
// friendPublicKey  the publicKey of the friend
func (s *StorageConn) getFriendDbId(friendPublicKey string) (int, error) {
	rows, err := s.db.Query("SELECT id FROM friends WHERE publicKey LIKE ?", friendPublicKey)
	if err != nil {
		log.Print("[persistence getFriendDbId] SELECT statement failed")
		return -1, err
	}
	defer rows.Close()

	if rows.Next() {
		var id int
		rows.Scan(&id)
		return id, nil
	} else {
		result, err := s.db.Exec(`INSERT INTO friends(publicKey) VALUES(?)`, friendPublicKey)
		if err != nil {
			log.Print("[persistence getFriendDbId] INSERT statement failed")
			return -1, err
		}
		id, err := result.LastInsertId()
		return int(id), err
	}
}
