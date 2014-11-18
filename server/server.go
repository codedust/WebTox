package main

import (
	"bufio"
	"crypto/tls"
	"golang.org/x/net/websocket"
	"io"
	"fmt"
	"net"
	"net/http"
)

func serveGUI() {
	var err error

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

	listener := &DowngradingListener{netListener, &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   "WebTox",
	}}

	mux := http.NewServeMux()
	mux.Handle("/events", websocket.Handler(handleWS))
	mux.HandleFunc("/api/", handleAPI)
	mux.HandleFunc("/", handleHTTP)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			// redirect HTTP to HTTPS
			r.URL.Host = r.Host
			r.URL.Scheme = "https"
			http.Redirect(w, r, r.URL.String(), http.StatusFound)
		} else {
			mux.ServeHTTP(w, r)
		}
	})

	err = http.Serve(listener, handler)
	if err != nil {
		panic(err)
	}
}

type DowngradingListener struct {
	net.Listener
	TLSConfig *tls.Config
}

type WrappedConnection struct {
	io.Reader
	net.Conn
}

func (l *DowngradingListener) Accept() (net.Conn, error) {
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

	wrapper := &WrappedConnection{br, conn}

	// 0x16 is the first byte of a TLS handshake
	if bs[0] == 0x16 {
		return tls.Server(wrapper, l.TLSConfig), nil
	}

	return wrapper, nil
}

func (c *WrappedConnection) Read(b []byte) (n int, err error) {
	return c.Reader.Read(b)
}
