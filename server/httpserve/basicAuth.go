package httpserve

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	pseudoRandom "math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	sessions    = make(map[string]bool)
	sessionsMut sync.Mutex
)

func BasicAuthHandler(next http.Handler, user string, password string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("sessionid")
		if err == nil && cookie != nil {
			sessionsMut.Lock()
			_, ok := sessions[cookie.Value]
			sessionsMut.Unlock()
			if ok {
				next.ServeHTTP(w, r)
				return
			}
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Basic ") {
			rejectWithHttpErrorNotAuthorized(w)
			return
		}

		authHeader = authHeader[len("Basic "):]
		authString, err := base64.StdEncoding.DecodeString(authHeader)
		if err != nil {
			rejectWithHttpErrorNotAuthorized(w)
			return
		}

		authFields := bytes.SplitN(authString, []byte(":"), 2)
		if len(authFields) != 2 {
			// no colon in authString
			rejectWithHttpErrorNotAuthorized(w)
			return
		}

		if string(authFields[0]) != user {
			rejectWithHttpErrorNotAuthorized(w)
			return
		}

		if sha512Sum(string(authFields[1])) != password {
			rejectWithHttpErrorNotAuthorized(w)
			return
		}

		sessionid, err := randomString(32)
		if err != nil {
			rejectWithHttpErrorNotAuthorized(w)
			return
		}
		sessionsMut.Lock()
		sessions[sessionid] = true
		sessionsMut.Unlock()
		http.SetCookie(w, &http.Cookie{
			Name:   "sessionid",
			Value:  sessionid,
			MaxAge: 0,
		})

		next.ServeHTTP(w, r)
	})
}

func rejectWithHttpErrorNotAuthorized(w http.ResponseWriter) {
	time.Sleep(time.Duration(pseudoRandom.Intn(100)+100) * time.Millisecond)
	w.Header().Set("WWW-Authenticate", "Basic realm=\"Authorization Required\"")
	http.Error(w, "Not Authorized", http.StatusUnauthorized)
}

func randomString(len int) (string, error) {
	bs := make([]byte, len)
	_, err := rand.Reader.Read(bs)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(bs), nil
}

func sha512Sum(s string) string {
	hasher := sha512.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}
