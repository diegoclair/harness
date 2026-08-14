package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// doerFunc adapts a func to the Doer interface.
type doerFunc func(req *http.Request) (*http.Response, error)

func (f doerFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func jsonResponse(status int, v any) *http.Response {
	b, _ := json.Marshal(v)
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(string(b))),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func TestBasic_ApplyAndBases(t *testing.T) {
	b := Basic{Email: "d@example.com", Token: "tok", Cloud: "lybel"}

	req, _ := http.NewRequest("GET", "https://x", nil)
	if err := b.Apply(req); err != nil {
		t.Fatal(err)
	}
	// base64("d@example.com:tok")
	if got := req.Header.Get("Authorization"); got != "Basic ZEBleGFtcGxlLmNvbTp0b2s=" {
		t.Errorf("Authorization = %q", got)
	}
	if b.ConfluenceBase() != "https://lybel.atlassian.net/wiki" {
		t.Errorf("ConfluenceBase = %q", b.ConfluenceBase())
	}
	if b.JiraBase() != "https://lybel.atlassian.net" {
		t.Errorf("JiraBase = %q", b.JiraBase())
	}
	if b.Kind() != ModeAPIToken {
		t.Errorf("Kind = %q", b.Kind())
	}
}

func oauthCreds() Credentials {
	return Credentials{
		Mode: ModeOAuth, ClientID: "cid", ClientSecret: "csec",
		AccessToken: "old-at", RefreshToken: "rt-1", CloudID: "cloud-uuid",
		Expiry: time.Now().Add(30 * time.Minute),
	}
}

func TestOAuth_Bases(t *testing.T) {
	o := NewOAuth(oauthCreds())
	if got := o.ConfluenceBase(); got != "https://api.atlassian.com/ex/confluence/cloud-uuid/wiki" {
		t.Errorf("ConfluenceBase = %q", got)
	}
	if got := o.JiraBase(); got != "https://api.atlassian.com/ex/jira/cloud-uuid" {
		t.Errorf("JiraBase = %q", got)
	}
}

// Links printed to users must point at the site, not the API gateway, which
// does not serve the web UI.
func TestOAuth_ConfluenceWebBase(t *testing.T) {
	creds := oauthCreds()
	creds.Site = "lybel"
	if got := NewOAuth(creds).ConfluenceWebBase(); got != "https://lybel.atlassian.net/wiki" {
		t.Errorf("ConfluenceWebBase = %q", got)
	}

	creds.Site = "" // pre-0.15 grant: degrade to the API base rather than a broken link
	o := NewOAuth(creds)
	if got := o.ConfluenceWebBase(); got != o.ConfluenceBase() {
		t.Errorf("without a site, ConfluenceWebBase = %q, want %q", got, o.ConfluenceBase())
	}
}

func TestOAuth_Apply_UsesValidTokenWithoutRefresh(t *testing.T) {
	o := NewOAuth(oauthCreds())
	o.Persist = false
	o.HTTP = doerFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatal("no HTTP call expected")
		return nil, nil
	})

	req, _ := http.NewRequest("GET", "https://x", nil)
	if err := o.Apply(req); err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer old-at" {
		t.Errorf("Authorization = %q", got)
	}
}

func TestOAuth_Apply_RefreshesExpiredAndRotates(t *testing.T) {
	redirectConfigDir(t)

	creds := oauthCreds()
	creds.Expiry = time.Now().Add(-time.Minute)
	if err := WriteCreds(creds); err != nil {
		t.Fatal(err)
	}

	var gotForm url.Values
	o := NewOAuth(creds)
	o.HTTP = doerFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != TokenURL {
			t.Errorf("refresh hit %q", req.URL)
		}
		body, _ := io.ReadAll(req.Body)
		gotForm, _ = url.ParseQuery(string(body))
		return jsonResponse(200, map[string]any{
			"access_token": "new-at", "refresh_token": "rt-2", "expires_in": 3600,
		}), nil
	})

	req, _ := http.NewRequest("GET", "https://x", nil)
	if err := o.Apply(req); err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer new-at" {
		t.Errorf("Authorization = %q", got)
	}
	if gotForm.Get("grant_type") != "refresh_token" || gotForm.Get("refresh_token") != "rt-1" ||
		gotForm.Get("client_id") != "cid" || gotForm.Get("client_secret") != "csec" {
		t.Errorf("refresh form = %v", gotForm)
	}

	// Rotation must be persisted: the file now holds rt-2.
	stored, err := ReadCreds()
	if err != nil {
		t.Fatal(err)
	}
	if stored.RefreshToken != "rt-2" || stored.AccessToken != "new-at" {
		t.Errorf("rotation not persisted: %+v", stored)
	}
}

// Grants on the bundled app store no secret; refresh must pull it from the
// binary, otherwise rotating the app secret would break every user until they
// logged in again.
func TestOAuth_ClientSecret_FallsBackToBundled(t *testing.T) {
	t.Setenv(clientSecretEnv, "bundled-secret")

	creds := oauthCreds()
	creds.ClientID = DefaultClientID
	creds.ClientSecret = ""
	if got := NewOAuth(creds).clientSecret(); got != "bundled-secret" {
		t.Errorf("clientSecret = %q, want the bundled one", got)
	}

	// A custom app must never silently borrow the bundled secret.
	creds.ClientID = "someone-elses-app"
	if got := NewOAuth(creds).clientSecret(); got != "" {
		t.Errorf("custom app with no stored secret = %q, want empty", got)
	}

	// A stored secret always wins.
	creds.ClientSecret = "stored"
	if got := NewOAuth(creds).clientSecret(); got != "stored" {
		t.Errorf("clientSecret = %q, want the stored one", got)
	}
}

// Sending an empty client_secret makes Atlassian reject the refresh, so the
// key must be absent, not blank.
func TestOAuth_Apply_RefreshOmitsEmptyClientSecret(t *testing.T) {
	redirectConfigDir(t)

	creds := oauthCreds()
	creds.ClientSecret = ""
	creds.Expiry = time.Now().Add(-time.Minute)
	if err := WriteCreds(creds); err != nil {
		t.Fatal(err)
	}

	var gotForm url.Values
	o := NewOAuth(creds)
	o.HTTP = doerFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		gotForm, _ = url.ParseQuery(string(body))
		return jsonResponse(200, map[string]any{
			"access_token": "new-at", "refresh_token": "rt-2", "expires_in": 3600,
		}), nil
	})

	req, _ := http.NewRequest("GET", "https://x", nil)
	if err := o.Apply(req); err != nil {
		t.Fatal(err)
	}
	if _, present := gotForm["client_secret"]; present {
		t.Errorf("client_secret must be omitted for a public client, got %v", gotForm)
	}
	if gotForm.Get("client_id") != "cid" || gotForm.Get("refresh_token") != "rt-1" {
		t.Errorf("refresh form = %v", gotForm)
	}
}

func TestOAuth_Apply_AdoptsSiblingRefresh(t *testing.T) {
	redirectConfigDir(t)

	// This process holds an expired token…
	stale := oauthCreds()
	stale.Expiry = time.Now().Add(-time.Minute)

	// …but a sibling process already refreshed and persisted a fresh one.
	fresh := stale
	fresh.AccessToken, fresh.RefreshToken = "sibling-at", "rt-9"
	fresh.Expiry = time.Now().Add(50 * time.Minute)
	if err := WriteCreds(fresh); err != nil {
		t.Fatal(err)
	}

	o := NewOAuth(stale)
	o.HTTP = doerFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatal("must adopt sibling token, not refresh")
		return nil, nil
	})

	req, _ := http.NewRequest("GET", "https://x", nil)
	if err := o.Apply(req); err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer sibling-at" {
		t.Errorf("Authorization = %q", got)
	}
}

func TestOAuth_Apply_InvalidGrantAsksForLogin(t *testing.T) {
	redirectConfigDir(t)

	creds := oauthCreds()
	creds.Expiry = time.Now().Add(-time.Minute)
	o := NewOAuth(creds)
	o.HTTP = doerFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(403, map[string]any{
			"error": "invalid_grant", "error_description": "unknown or expired refresh token",
		}), nil
	})

	req, _ := http.NewRequest("GET", "https://x", nil)
	err := o.Apply(req)
	if err == nil || !strings.Contains(err.Error(), "run `login` again") {
		t.Errorf("want re-login guidance, got %v", err)
	}
}

func TestResolve_EnvVarsWinOverFile(t *testing.T) {
	redirectConfigDir(t)
	if err := WriteCreds(oauthCreds()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ATLASSIAN_EMAIL", "env@example.com")
	t.Setenv("ATLASSIAN_API_TOKEN", "env-tok")

	a, err := Resolve(Options{Cloud: "lybel"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Kind() != ModeAPIToken {
		t.Errorf("env creds must resolve to Basic, got %q", a.Kind())
	}
}

func TestResolve_OAuthFromFile(t *testing.T) {
	redirectConfigDir(t)
	os.Unsetenv("ATLASSIAN_EMAIL")
	os.Unsetenv("ATLASSIAN_API_TOKEN")
	if err := WriteCreds(oauthCreds()); err != nil {
		t.Fatal(err)
	}

	// Cloud absent: OAuth must not require it.
	a, err := Resolve(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Kind() != ModeOAuth {
		t.Errorf("Kind = %q, want oauth", a.Kind())
	}
}

func TestResolve_ModeAPITokenOverridesStoredOAuth(t *testing.T) {
	redirectConfigDir(t)
	c := oauthCreds()
	c.Mode = ModeAPIToken
	c.Email, c.Token = "d@example.com", "tok"
	if err := WriteCreds(c); err != nil {
		t.Fatal(err)
	}

	a, err := Resolve(Options{Cloud: "lybel"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Kind() != ModeAPIToken {
		t.Errorf("auth_mode=apitoken must force Basic, got %q", a.Kind())
	}
}

func TestResolve_LegacyFallbackWarns(t *testing.T) {
	redirectConfigDir(t)
	legacyDir := t.TempDir()
	legacy := legacyDir + "/credentials"
	if err := os.WriteFile(legacy, []byte("email=old@example.com\ntoken=old-tok\n"), 0600); err != nil {
		t.Fatal(err)
	}

	var warn strings.Builder
	a, err := Resolve(Options{Cloud: "lybel", Stderr: &warn, LegacyCredsPaths: []string{legacy}})
	if err != nil {
		t.Fatal(err)
	}
	if a.Kind() != ModeAPIToken {
		t.Errorf("Kind = %q", a.Kind())
	}
	if !strings.Contains(warn.String(), "legacy path") {
		t.Errorf("expected migration warning, got %q", warn.String())
	}
}

func TestResolve_NoCredsErrorMentionsBothPaths(t *testing.T) {
	redirectConfigDir(t)
	_, err := Resolve(Options{Cloud: "lybel"})
	if err == nil {
		t.Fatal("want error")
	}
	for _, want := range []string{"login", "setup"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}
}

func TestResolve_BasicWithoutCloudErrors(t *testing.T) {
	redirectConfigDir(t)
	t.Setenv("ATLASSIAN_EMAIL", "e@x.c")
	t.Setenv("ATLASSIAN_API_TOKEN", "t")

	_, err := Resolve(Options{})
	if err == nil || !strings.Contains(err.Error(), "subdomain") {
		t.Errorf("want cloud-subdomain error, got %v", err)
	}
}

func TestRequestToken_ErrorEnvelope(t *testing.T) {
	client := doerFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(400, map[string]any{"error": "invalid_request", "error_description": "boom"}), nil
	})
	_, err := requestToken(client, url.Values{"grant_type": {"authorization_code"}})
	te, ok := err.(*tokenError)
	if !ok {
		t.Fatalf("want *tokenError, got %T: %v", err, err)
	}
	if te.Code != "invalid_request" || te.StatusCode != 400 {
		t.Errorf("tokenError = %+v", te)
	}
}

func TestRequestToken_MissingAccessToken(t *testing.T) {
	client := doerFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, map[string]any{"expires_in": 3600}), nil
	})
	_, err := requestToken(client, url.Values{})
	if err == nil || !strings.Contains(err.Error(), "missing access_token") {
		t.Errorf("got %v", err)
	}
}
