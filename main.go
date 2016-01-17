package main

import (
	"encoding/base64"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/th3osmith/greader/pure"
	"github.com/th3osmith/greader/test"
	"log"
	"net/http"
	"strings"
	"time"
)

func Serve() {
	http.Handle("/test", loggingHandler(http.HandlerFunc(testHandler)))
	http.Handle("/panic", loggingHandler(recoverHandler(http.HandlerFunc(panicHandler))))
	http.HandleFunc("/websocket", websocketEchoHandler)

	// Pure Test
	mux := pure.NewPureMux()
	h := pure_test.NewPureHandler()
	mux.RegisterHandler("data", h)

	http.Handle("/pure", pure.WebsocketHandler(*mux))

	yes := singleUserAuthenticator{"tata", "yoyo"}
	http.Handle("/protected", loggingHandler(recoverHandler(yes.authHandler(http.HandlerFunc(testHandler)))))
	http.ListenAndServe(":3000", nil)
}

func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		w2 := NewResponseWriter(w)

		next.ServeHTTP(w2, r)

		t2 := time.Now()

		log.Printf("[%s] %d %q %v\n", r.Method, w2.Status(), r.URL.String(), t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic Recovery: %+v", err)
				code := http.StatusInternalServerError
				http.Error(w, http.StatusText(code), code)
			}
		}()

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func websocketEchoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Fprintf(w, "Upgrade failed: %v", err)
		return
	}

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[WebSocket] Error Reading message: %v\n", err)
			continue
		}
		if err = conn.WriteMessage(messageType, p); err != nil {
			log.Printf("[WebSocket] Error Eriting message %v (%v): %v\n", string(p), messageType, err)
			continue
		}
	}

}

type Authenticator interface {
	checkAuth(credentials string) bool
	authHandler(next http.Handler) http.Handler
}

type singleUserAuthenticator struct {
	user     string
	password string
}

func (y *singleUserAuthenticator) checkAuth(credentials string) bool {

	usernamePassword := strings.Split(credentials, ":")

	if len(usernamePassword) != 2 {
		return false
	}

	if usernamePassword[0] == y.user && usernamePassword[1] == y.password {
		return true
	}

	return false
}

func (y *singleUserAuthenticator) authHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		auth := r.Header.Get("Authorization")

		if !strings.HasPrefix(auth, "Basic ") {
			AuthRequest(w)
			return
		}

		credentials, err := base64.StdEncoding.DecodeString(auth[6:])

		if err != nil || y.checkAuth(string(credentials)) != true {
			AuthRequest(w)
			return
		}

		next.ServeHTTP(w, r)

	}

	return http.HandlerFunc(fn)
}

func checkAuth(credentials string) bool {
	return true
}

func AuthRequest(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Authentication Required"`)
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, "Authentication required")
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTeapot)
	fmt.Fprintf(w, "Test")
}

func panicHandler(w http.ResponseWriter, r *http.Request) {
	panic("Oulala")
	fmt.Fprintf(w, "Test")
}

func main() {
	Serve()
}

type MyResponseWriter interface {
	http.ResponseWriter
	Status() int
}

type myResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (w *myResponseWriter) WriteHeader(code int) {
	w.StatusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *myResponseWriter) Status() int {
	return w.StatusCode
}

func NewResponseWriter(w http.ResponseWriter) MyResponseWriter {
	return &myResponseWriter{w, 200}
}

// WebSocket Test

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
