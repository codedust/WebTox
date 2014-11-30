package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
)

func serveGUI() {
	// TODO support 0.0.0.0 and different ports
	netListener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("[serveGUI] Could not bind server to port")
		panic(err)
	}

	// load the certificate
	cert, err := loadCertificate(CFG_DATA_DIR, CFG_CERT_PREFIX)
	if err != nil {
		fmt.Println("[serveGUI] Could not load certificate")
		panic(err)
	}

	// create a listener that can listen to encrypted and unencrypted connections
	listener := &muxListener{netListener, &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   "WebTox",
	}}

	mux := http.NewServeMux()
	mux.Handle("/events", handleWS)
	mux.Handle("/api/", handleAPI)
	mux.Handle("/", http.FileServer(http.Dir("../html")))

	// add authentication
	// TODO: no auth if no password is set
	handleAuth := basicAuthHandler(mux)

	// redirect from http to https
	handleRedirect := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			r.URL.Host = r.Host
			r.URL.Scheme = "https"
			http.Redirect(w, r, r.URL.String(), http.StatusFound)
		} else {
			handleAuth.ServeHTTP(w, r)
		}
	})

	err = http.Serve(listener, handleRedirect)
	if err != nil {
		panic(err)
	}
}

type muxListener struct {
	net.Listener
	TLSConfig *tls.Config
}

type wrappedConnection struct {
	io.Reader
	net.Conn
}

// overrides muxListener.Accept to handle tls and non-tls connections
func (l *muxListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	br := bufio.NewReader(conn)
	bs, err := br.Peek(1)
	if err != nil {
		// We hit a read error here, but the Accept() call succeeded so we must not return an error.
		// We return the connection as is and let whoever tries to use it deal with the error.
		return conn, nil
	}

	wrapper := &wrappedConnection{br, conn}

	// 0x16 is the first byte of a TLS handshake
	if bs[0] == 0x16 {
		return tls.Server(wrapper, l.TLSConfig), nil
	}

	return wrapper, nil
}

func (c *wrappedConnection) Read(b []byte) (n int, err error) {
	return c.Reader.Read(b)
}
