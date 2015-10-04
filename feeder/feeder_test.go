package feeder_test

import (
	"encoding/base64"
	"fmt"
	"github.com/th3osmith/greader/feeder"
	"github.com/th3osmith/rss"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

var counter int

func TestSeenSlice(t *testing.T) {

	control := []string{"0", "1", "2", "3", "4"}
	s := []string{"2", "3", "4", "5", "6"}

	s = feeder.AddNew(s, "0", "1")

	for i, el := range control {
		if s[i] != el {
			t.Error("AddNew Error.", s)
		}
	}

}

func TestSubscription(t *testing.T) {

	feed, err := feeder.NewFeed("http://localhost:3000/hn")
	if err != nil {
		t.Error(err)
	}

	sub := new(feeder.TestSubscriber)
	feed.Register(sub)

	feed.Clear()
	feed.Update(true)

	if len(sub.Items) < 20 {
		t.Error("Subscriber did not get all items.", len(sub.Items))
	} else if len(sub.Items) > 20 {
		t.Error("Pb with clear")
	}

	control := []string{"http://www.itworld.com/article/2987438/newly-found-truecrypt-flaw-allows-full-system-compromise.html", "https://answers.microsoft.com/en-us/windows/forum/windows_7-update/windows-7-update-appears-to-be-compromised/e96a0834-a9e9-4f03-a187-bef8ee62725e", "https://itunes.apple.com/app/os-x-el-capitan/id1018109117?mt=12", "http://okmij.org/ftp/Computation/IO-monad-history.html"}

	// Testing Seen
	for idx, el := range control {
		if feed.Seen[idx] != el {
			t.Error("Error while registering Seen.", feed.Seen)
		}
	}

}

func TestSeed(t *testing.T) {

	rss.CacheParsedItemIDs(true)

	feed, err := feeder.NewAuthFeed("http://localhost:3000/auth/hn", "username", "password")
	if err != nil {
		t.Error(err)
	}

	feed.ReadNew()
	seed := feed.ExportSeed()

	feedA, err := feeder.NewFeedFromSeed(seed)
	if err != nil {
		t.Error(err)
	}

	sub := new(feeder.TestSubscriber)
	feedA.Register(sub)

	feedA.Update(true)
	feedA.Update(true)

	if len(sub.Items) != 20 {
		t.Error("History export Not working", len(sub.Items))
	}

	rss.CacheParsedItemIDs(false)

}

func TestFeed(t *testing.T) {

	_, err := feeder.NewFeed("http://localhost:3000/auth/hn")
	if err != nil && err.Error() != "HTTP Error. Status Code 401" {
		t.Error("Error badly handeled")
	}

	_, err = feeder.NewAuthFeed("http://localhost:3000/auth/hn", "username", "password")
	if err != nil && err.Error() != "Impossible to parse Feed. EOF" {
		t.Error("Auth not working.", err)
	}

	feed, err := feeder.NewFeed("http://localhost:3000/hn")
	if err != nil {
		t.Error(err)
	}

	if feed.Url != "http://localhost:3000/hn" ||
		feed.Status != feeder.StatusOK ||
		feed.Name != "Hacker News" {

		t.Log(feed)
		t.Error("Error during Feed Init")

	}

}

func TestMain(m *testing.M) {
	rss.CacheParsedItemIDs(false)
	counter = 0
	go startServer()
	os.Exit(m.Run())
}

/*
 * HTTP Server
 */

func startServer() {
	http.Handle("/hn", http.HandlerFunc(hnHandler))
	http.Handle("/auth/hn", authHandler(http.HandlerFunc(hnHandler)))
	http.Handle("/e500", http.HandlerFunc(errorHandler))
	http.Handle("/e404", http.HandlerFunc(notFoundHandler))
	http.ListenAndServe(":3000", nil)
}

// Each time you go to the url serve a more recent file
func hnHandler(w http.ResponseWriter, r *http.Request) {
	file := fmt.Sprintf("testdata/hn_%d", counter)
	counter = (counter + 1) % 2

	f, err := os.Open(file)
	if err != nil {
		panic("Unable to load Test file")
	}

	io.Copy(w, f)
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Error 500", http.StatusInternalServerError)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Error 404", http.StatusNotFound)
}

func authHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		auth := r.Header.Get("Authorization")

		if !strings.HasPrefix(auth, "Basic ") {
			w.Header().Set("WWW-Authenticate", `Basic realm="Authentication Required"`)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Error 401")
			return
		}

		credentials, err := base64.StdEncoding.DecodeString(auth[6:])

		if err != nil || checkAuth(string(credentials)) != true {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "Error 403")
			return
		}

		next.ServeHTTP(w, r)

	}

	return http.HandlerFunc(fn)
}

func checkAuth(credentials string) bool {

	usernamePassword := strings.Split(credentials, ":")

	if len(usernamePassword) != 2 {
		return false
	}

	if usernamePassword[0] == "username" && usernamePassword[1] == "password" {
		return true
	}

	return false
}
