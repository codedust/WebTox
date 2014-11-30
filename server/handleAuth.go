package main

import (
	"bytes"
	"encoding/base64"
	pseudoRandom "math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	sessions     = make(map[string]bool)
	sessionsMut  sync.Mutex
	AuthUser     string = "user"
	AuthPassword string = "5b722b307fce6c944905d132691d5e4a2214b7fe92b738920eb3fce3a90420a19511c3010a0e7712b054daef5b57bad59ecbd93b3280f210578f547f4aed4d25" // "pass"
	// TODO store password in db and allow the user to change it
)

func basicAuthHandler(next http.Handler) http.Handler {
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

		if string(authFields[0]) != AuthUser {
			rejectWithHttpErrorNotAuthorized(w)
			return
		}

		if sha512Sum(string(authFields[1])) != AuthPassword {
			rejectWithHttpErrorNotAuthorized(w)
			return
		}

		sessionid := randomString(32)
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
