package httpserve

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
)

func ListenAndUpgradeToHTTPS(addr string, certFile string, keyFile string, handler http.Handler) error {
	netListener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	// create a listener that can listen to encrypted and unencrypted connections
	listener := &muxListener{netListener, &tls.Config{
		Certificates: []tls.Certificate{cert},
	}}

	// redirect from http to https
	handleRedirect := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			r.URL.Host = r.Host
			r.URL.Scheme = "https"
			http.Redirect(w, r, r.URL.String(), http.StatusFound)
		} else {
			handler.ServeHTTP(w, r)
		}
	})

	err = http.Serve(listener, handleRedirect)
	if err != nil {
		return err
	}

	return nil
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
