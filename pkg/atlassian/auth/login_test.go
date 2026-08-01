package auth

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSubdomainFromURL(t *testing.T) {
	cases := map[string]string{
		"https://lybel.atlassian.net":  "lybel",
		"https://my-co.atlassian.net/": "my-co",
		"not a url at all\x7f":         "",
	}
	for in, want := range cases {
		if got := subdomainFromURL(in); got != want {
			t.Errorf("subdomainFromURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPickResource(t *testing.T) {
	resources := []accessibleResource{
		{ID: "1", Name: "Lybel", URL: "https://lybel.atlassian.net"},
		{ID: "2", Name: "Other", URL: "https://other.atlassian.net"},
	}

	// --site match by URL fragment.
	r, err := pickResource(resources, "other", nil, io.Discard)
	if err != nil || r.ID != "2" {
		t.Errorf("site match: got %+v, %v", r, err)
	}

	// --site with no match errors.
	if _, err := pickResource(resources, "nope", nil, io.Discard); err == nil {
		t.Error("want error for unmatched --site")
	}

	// Single resource short-circuits.
	r, err = pickResource(resources[:1], "", nil, io.Discard)
	if err != nil || r.ID != "1" {
		t.Errorf("single: got %+v, %v", r, err)
	}

	// Interactive choice.
	r, err = pickResource(resources, "", bufio.NewReader(strings.NewReader("2\n")), io.Discard)
	if err != nil || r.ID != "2" {
		t.Errorf("interactive: got %+v, %v", r, err)
	}

	// Empty answer defaults to 1.
	r, err = pickResource(resources, "", bufio.NewReader(strings.NewReader("\n")), io.Discard)
	if err != nil || r.ID != "1" {
		t.Errorf("default: got %+v, %v", r, err)
	}
}

func TestRandomState_Unique(t *testing.T) {
	a, err := randomState()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := randomState()
	if a == b || len(a) != 32 {
		t.Errorf("states not unique/sized: %q %q", a, b)
	}
}

// callbackURL builds the loopback callback with the given query.
func callbackURL(port int, query string) string {
	return fmt.Sprintf("http://127.0.0.1:%d/callback?%s", port, query)
}

func TestWaitForCallback_HappyPath(t *testing.T) {
	const port = 18517
	state := "st4te"

	type out struct {
		code string
		err  error
	}
	done := make(chan out, 1)
	go func() {
		code, err := waitForCallback("http://unused", state, port, true, io.Discard, io.Discard)
		done <- out{code, err}
	}()

	// Poll until the listener answers, then deliver the redirect.
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get(callbackURL(port, "state="+state+"&code=the-code"))
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("callback server never came up: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	select {
	case r := <-done:
		if r.err != nil || r.code != "the-code" {
			t.Errorf("got code=%q err=%v", r.code, r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("waitForCallback did not return")
	}
}

func TestWaitForCallback_StateMismatch(t *testing.T) {
	const port = 18518

	type out struct {
		code string
		err  error
	}
	done := make(chan out, 1)
	go func() {
		code, err := waitForCallback("http://unused", "expected", port, true, io.Discard, io.Discard)
		done <- out{code, err}
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get(callbackURL(port, "state=WRONG&code=x"))
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("callback server never came up: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	select {
	case r := <-done:
		if r.err == nil || !strings.Contains(r.err.Error(), "state mismatch") {
			t.Errorf("want state-mismatch error, got code=%q err=%v", r.code, r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("waitForCallback did not return")
	}
}

func TestWaitForCallback_DeniedAuthorization(t *testing.T) {
	const port = 18519

	done := make(chan error, 1)
	go func() {
		_, err := waitForCallback("http://unused", "s", port, true, io.Discard, io.Discard)
		done <- err
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get(callbackURL(port, "error=access_denied&error_description=nope"))
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("callback server never came up: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "access_denied") {
			t.Errorf("want denial error, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("waitForCallback did not return")
	}
}

func TestNewPKCE(t *testing.T) {
	verifier, challenge, err := newPKCE()
	if err != nil {
		t.Fatal(err)
	}
	// RFC 7636: 43–128 chars, base64url without padding.
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("verifier length %d out of range", len(verifier))
	}
	if strings.ContainsAny(verifier+challenge, "=+/") {
		t.Errorf("verifier/challenge must be base64url unpadded: %q %q", verifier, challenge)
	}
	sum := sha256.Sum256([]byte(verifier))
	if want := base64.RawURLEncoding.EncodeToString(sum[:]); challenge != want {
		t.Errorf("challenge = %q, want S256 of verifier (%q)", challenge, want)
	}

	other, _, err := newPKCE()
	if err != nil {
		t.Fatal(err)
	}
	if other == verifier {
		t.Error("verifier must differ across calls")
	}
}

func TestRunLogin_PrintRedirectURI(t *testing.T) {
	var out strings.Builder
	code, err := RunLogin([]string{"--print-redirect-uri"}, strings.NewReader(""), &out, io.Discard)
	if err != nil || code != exitOK {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if strings.TrimSpace(out.String()) != "http://localhost:8517/callback" {
		t.Errorf("redirect uri = %q", out.String())
	}
}

func TestRunLogin_UnknownFlag(t *testing.T) {
	code, err := RunLogin([]string{"--bogus"}, strings.NewReader(""), io.Discard, io.Discard)
	if err == nil || code != exitErr {
		t.Errorf("code=%d err=%v", code, err)
	}
}
