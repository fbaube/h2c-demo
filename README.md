## Demo of HTTP/2 Cleartext (H2C) in golang

This repo is a clone of [another repo]
(https://github.com/thrawn01/h2c-golang-example)
discussed in this [very helpful article]
(https://medium.com/@thrawn01/http-2-cleartext-h2c-client-example-in-go-8167c7a4181e).

I had problems getting DIY certs to work with HTTP/2, which mandates HTTPS.
Well, unless you use h2c (which means "h2, but with cleartext"), in which
case you connect in cleartext (i.e. HTTP) and it upgrades - via sufficient
strickery - to HTTP/2.

### Tech Overview

As described [here](https://my.f5.com/manage/s/article/K47440400), h2c has
two elements:

1) **Upgrade** from HTTP/1.1: <br/>
When a client has no prior knowledge about a server's h2c support, it
makes a request to an HTTP URI in HTTP/1.1 and includes an `Upgrade`
header field with the h2c token, i.e. `Upgrade: h2c`. A server that
supports HTTP/2 responds with an HTTP/1.1 `101` (Switching Protocols)
response and _Hey, presto!_ the exchange proceeds in HTTP/2.

2) **Prior knowledge** that a server supports h2c: <br/>
The client initiates HTTP/2 messages directly after the
TCP handshake without an initial HTTP/1.1 exchange.

Standard golang code supports HTTP2 but does not directly support
H2C; H2C support lies in package `golang.org/x/net/http2/h2c`.

To make your HTTP server H2C-capable (both Upgrade and Prior knowledge),
in addition to the standard support for HTTP/2 and HTTP/1.1, wrap your
handler or mux with `h2c.NewHandler()` like so:

```go
h2s := &http2.Server{}

handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, %v, http: %v", r.URL.Path, r.TLS == nil)
})

server := &http.Server{
    Addr:    "0.0.0.0:1010",
    Handler: h2c.NewHandler(handler, h2s),
}

fmt.Printf("Listening [0.0.0.0:1010]...\n")
checkErr(server.ListenAndServe(), "while listening")
```

If you don't care about supporting **Upgrade** for HTTP/1.1 then
you can run this code which only supports **Prior knowledge**:

```go
server := http2.Server{}

l, err := net.Listen("tcp", "0.0.0.0:1010")
checkErr(err, "while listening")

fmt.Printf("Listening [0.0.0.0:1010]...\n")
for {
    conn, err := l.Accept()
    checkErr(err, "during accept")

    server.ServeConn(conn, &http2.ServeConnOpts{
        Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            fmt.Fprintf(w, "Hello, %v, http: %v", r.URL.Path, r.TLS == nil)
        }),
    })
}
```

### Testing your server

Once you have a running server you can test your server by
installing `curl-openssl`: 

```
$ brew install curl-openssl  # on macos 

# Prepend curl-openssl to your path
$ export PATH="/usr/local/opt/curl-openssl/bin:$PATH"
```

You can now use curl to test your H2C enabled server like so.

<big> Connect via HTTP1.1 (curl: **`--http2`**)
and then **Upgrade** to HTTP/2 (H2C) </big>
```
curl -v --http2 http://localhost:1010
*   Trying ::1:1010...
* TCP_NODELAY set
* Connected to localhost (::1) port 1010 (#0)
> GET / HTTP/1.1
> Host: localhost:1010
> User-Agent: curl/7.65.0
> Accept: */*
> Connection: Upgrade, HTTP2-Settings
> Upgrade: h2c
> HTTP2-Settings: AAMAAABkAARAAAAAAAIAAAAA
>
* Mark bundle as not supporting multiuse
< HTTP/1.1 101 Switching Protocols
< Connection: Upgrade
< Upgrade: h2c
* Received 101
* Using HTTP2, server supports multi-use
* Connection state changed (HTTP/2 confirmed)
* Copying HTTP/2 data in stream buffer to connection buffer after upgrade: len=0
* Connection state changed (MAX_CONCURRENT_STREAMS == 250)!
< HTTP/2 200
< content-type: text/plain; charset=utf-8
< content-length: 20
< date: Wed, 05 Jun 2019 19:01:40 GMT
<
* Connection #0 to host localhost left intact
Hello, /, http: true
```

<big> Using **Prior knowledge**, connect via HTTP/2
(curl: **`--http2-prior-knowledge`**) </big>
```
curl -v --http2-prior-knowledge http://localhost:1010
*   Trying ::1:1010...
* TCP_NODELAY set
* Connected to localhost (::1) port 1010 (#0)
* Using HTTP2, server supports multi-use
* Connection state changed (HTTP/2 confirmed)
* Copying HTTP/2 data in stream buffer to connection buffer after upgrade: len=0
* Using Stream ID: 1 (easy handle 0x7fdab8007000)
> GET / HTTP/2
> Host: localhost:1010
> User-Agent: curl/7.65.0
> Accept: */*
>
* Connection state changed (MAX_CONCURRENT_STREAMS == 250)!
< HTTP/2 200
< content-type: text/plain; charset=utf-8
< content-length: 20
< date: Wed, 05 Jun 2019 19:00:43 GMT
<
* Connection #0 to host localhost left intact
Hello, /, http: true
```

Remember the statement above that the golang standard library does not
support H2C ? It is technically correct but there is a workaround to get
the golang standard http2 client to connect to an H2C enabled server.

To do so, override `DialTLS` and set the supersecret flag `AllowHTTP`:

```go
client := http.Client{
    Transport: &http2.Transport{
        // So http2.Transport won't complain that the protocol isn't 'https'
        AllowHTTP: true,
        // Pretend we are dialing a TLS endpoint.
	// (Note, we ignore the passed tls.Config)
        DialTLSContext: func(ctx context.Context, network,
			addr string, cfg *tls.Config) (net.Conn, error) {
            var d net.Dialer
            return d.DialContext(ctx, network, addr)
        },
    },
}

resp, _ := client.Get(url)
fmt.Printf("Client Proto: %d\n", resp.ProtoMajor)
```

This all looks a bit dodgy but actually works well in production.

