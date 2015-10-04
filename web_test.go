package main_test

import (
	"bytes"
	"github.com/th3osmith/greader"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
)

var buf bytes.Buffer

func TestListening(t *testing.T) {

	resp, err := http.Get("http://localhost:3000/test")
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusTeapot {
		t.Error("Error Setting/Getting the Status Code")
	}

	if output := buf.String(); !strings.Contains(output, "[GET] 418 \"/test\"") {
		t.Error("Invalid Log output", output)
	}
	buf.Reset()

	t.Logf(buf.String())

}

func TestPanicRecover(t *testing.T) {

	_, err := http.Get("http://localhost:3000/panic")
	if err != nil {
		t.Error(err)
	}
}

func TestAuth(t *testing.T) {

	resp, err := http.Get("http://localhost:3000/protected")
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Error("Bad Status", resp.StatusCode)
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:3000/protected", nil)
	req.SetBasicAuth("tata", "yoyo")

	reqA, err := http.NewRequest("GET", "http://localhost:3000/protected", nil)
	reqA.SetBasicAuth("toto", "yoyo")

	resp, err = client.Do(req)
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusTeapot {
		t.Error("Bad Status", resp.StatusCode)
	}

	resp, err = client.Do(reqA)
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Error("Bad Status", resp.StatusCode)
	}

}

func TestMain(m *testing.M) {

	// Test Setup
	go main.Serve()
	log.SetOutput(&buf)

	os.Exit(m.Run())

}
