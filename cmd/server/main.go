package main

import (
	"fmt"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net"
	"net/http"
	"os"
)

func checkErr(err error, msg string) {
	if err == nil {
		return
	}
	fmt.Printf("Fatal error: %s: %s \n", msg, err)
	os.Exit(1)
}

func main() {
	H2CServerUpgrade() // understands both HTTP 1.1 and HTTP/2 requests 
	//H2CServerPrior() // understands only HTTP/2 requests 
}

// https://pkg.go.dev/net/http#HandlerFunc
// type HandlerFunc func(ResponseWriter, *Request)
// This type is an adapter to allow the use of ordinary functions as 
// HTTP handlers. If f is a function with the appropriate signature,
// HandlerFunc(f) is a Handler that calls f.

var theHandler http.HandlerFunc =
    http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Handler got new connection: << %+v >>\n", r)
	fmt.Fprintf(w, "Hi, <%v>! Secure cnxn? <%t>\n", r.URL.Path, r.TLS != nil)
	})

// This server supports "H2C upgrade" and "H2C prior knowledge" along 
// with standard HTTP/2 and HTTP/1.1 that golang natively supports.
func H2CServerUpgrade() {
	h2s := &http2.Server{}
	server := &http.Server{
		Addr:    "0.0.0.0:1010",
		Handler: h2c.NewHandler(theHandler, h2s),
	}
	fmt.Printf("Listening [0.0.0.0:1010]...\n")
	checkErr(server.ListenAndServe(), "while listening")
}

// This server only supports "H2C prior knowledge".
// You can add standard HTTP/2 support by adding a TLS config.
func H2CServerPrior() {
	server := http2.Server{}
	l, err := net.Listen("tcp", "0.0.0.0:1010")
	checkErr(err, "while listening")
	fmt.Printf("Listening [0.0.0.0:1010]...\n")
	for {
		conn, err := l.Accept()
		checkErr(err, "during accept")
		server.ServeConn(conn, &http2.ServeConnOpts{
			Handler: theHandler, 
		})
	}
}
