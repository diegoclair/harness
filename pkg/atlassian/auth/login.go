package auth

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// AuthorizeURL is Atlassian's OAuth authorization endpoint. Overridable in tests.
var AuthorizeURL = "https://auth.atlassian.com/authorize"

// DefaultClientID is the shared "Lybel Skills" OAuth app, so users get a
// browser login with nothing to register. Override with --client-id (plus
// --client-secret) to use your own app.
const DefaultClientID = "ypsZAoCPwWzcBwxbbxC72g6JyjCkTRqT"

// DefaultClientSecret pairs with DefaultClientID. Atlassian rejects the PKCE
// token exchange without a secret (OAUTH20-2491), so a shipped CLI has no way
// to be a true public client; the secret only identifies the app, never the
// user. It is injected at release build time
// (-X …/auth.DefaultClientSecret=…) and stays empty in the source tree, which
// keeps it out of git and away from secret scanners. Builds without it fall
// back to --client-secret / the env var.
var DefaultClientSecret string

// clientSecretEnv lets local builds supply the app secret without a rebuild.
const clientSecretEnv = "ATLASSIAN_OAUTH_CLIENT_SECRET"

// DefaultPort is the local callback port. It must match the callback URL
// registered in the user's 3LO app (http://localhost:8517/callback).
const DefaultPort = 8517

// DefaultScopes covers everything the confluence-docs and jira-tickets CLIs
// call today. Confluence v2 endpoints require granular scopes; Jira v3 uses
// classic ones. offline_access is what makes Atlassian issue a refresh token.
// Adjust with `login --scopes` if your app grants a different set.
var DefaultScopes = strings.Join([]string{
	"offline_access",
	// Confluence (granular; v2 API requires these)
	"read:page:confluence",
	"write:page:confluence",
	"read:space:confluence",
	"read:label:confluence",
	"write:label:confluence",
	"read:content:confluence",
	"write:content:confluence",
	"read:content-details:confluence",
	"read:content.metadata:confluence",
	"read:user:confluence",
	// Jira (classic)
	"read:jira-work",
	"write:jira-work",
	"read:jira-user",
}, " ")

// loginTimeout bounds how long we wait for the browser round-trip.
const loginTimeout = 5 * time.Minute

// Exit codes, aligned with the setup package.
const (
	exitOK      = 0
	exitErr     = 1
	exitAuthErr = 2
	exitNetErr  = 3
)

// accessibleResource is one entry of oauth/token/accessible-resources.
type accessibleResource struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Scopes []string `json:"scopes"`
}

// RunLogin implements the `login` sub-command: a browser-based OAuth 2.0
// (3LO) flow against the user's own Atlassian app (BYOA — bring your own
// app). On success the grant is stored in the shared credentials file and
// auth_mode switches to oauth.
func RunLogin(args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	var (
		clientID     string
		clientSecret string
		scopes       string
		site         string
		port         = DefaultPort
		noBrowser    bool
		printRedirect bool
	)

	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func(name string) (string, bool) {
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "flag %s requires a value\n", name)
				return "", false
			}
			i++
			return args[i], true
		}
		var ok bool
		switch {
		case a == "--client-id":
			clientID, ok = next(a)
			if !ok {
				return exitErr, fmt.Errorf("missing value")
			}
		case strings.HasPrefix(a, "--client-id="):
			clientID = a[len("--client-id="):]
		case a == "--client-secret":
			clientSecret, ok = next(a)
			if !ok {
				return exitErr, fmt.Errorf("missing value")
			}
		case strings.HasPrefix(a, "--client-secret="):
			clientSecret = a[len("--client-secret="):]
		case a == "--scopes":
			scopes, ok = next(a)
			if !ok {
				return exitErr, fmt.Errorf("missing value")
			}
		case strings.HasPrefix(a, "--scopes="):
			scopes = a[len("--scopes="):]
		case a == "--site":
			site, ok = next(a)
			if !ok {
				return exitErr, fmt.Errorf("missing value")
			}
		case strings.HasPrefix(a, "--site="):
			site = a[len("--site="):]
		case a == "--no-browser":
			noBrowser = true
		case a == "--print-redirect-uri":
			printRedirect = true
		case a == "--help" || a == "-h":
			printLoginHelp(stdout)
			return exitOK, nil
		default:
			fmt.Fprintf(stderr, "unknown flag: %s\n", a)
			printLoginHelp(stderr)
			return exitErr, fmt.Errorf("unknown flag")
		}
	}

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)
	if printRedirect {
		fmt.Fprintln(stdout, redirectURI)
		return exitOK, nil
	}

	// Prefill from the store so re-login reuses a custom app, then fall back
	// to the built-in public client.
	stored, _ := ReadCreds()
	if clientID == "" {
		clientID = stored.ClientID
	}
	if clientSecret == "" && clientID == stored.ClientID {
		clientSecret = stored.ClientSecret
	}
	if clientID == "" {
		clientID = DefaultClientID
	}
	if clientSecret == "" && clientID == DefaultClientID {
		clientSecret = builtinClientSecret()
		if clientSecret == "" {
			fmt.Fprintf(stderr, "this build has no bundled app secret — set %s or pass --client-id/--client-secret for your own app\n", clientSecretEnv)
			return exitErr, fmt.Errorf("no client secret available")
		}
	}
	if scopes == "" {
		if stored.Scopes != "" {
			scopes = stored.Scopes
		} else {
			scopes = DefaultScopes
		}
	}

	reader := bufio.NewReader(stdin)

	state, err := randomState()
	if err != nil {
		return exitErr, err
	}
	verifier, challenge, err := newPKCE()
	if err != nil {
		return exitErr, err
	}

	authURL := AuthorizeURL + "?" + url.Values{
		"audience":              {"api.atlassian.com"},
		"client_id":             {clientID},
		"scope":                 {scopes},
		"redirect_uri":          {redirectURI},
		"state":                 {state},
		"response_type":         {"code"},
		"prompt":                {"consent"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}.Encode()

	code, err := waitForCallback(authURL, state, port, noBrowser, stdout, stderr)
	if err != nil {
		return exitAuthErr, err
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	tr, err := requestToken(httpClient, form)
	if err != nil {
		fmt.Fprintf(stderr, "token exchange failed: %v\n", err)
		return exitAuthErr, err
	}
	if tr.RefreshToken == "" {
		fmt.Fprintln(stderr, "warning: no refresh token returned — make sure the offline_access scope is in the app AND in --scopes; without it the session dies in 1 hour")
	}

	resources, err := fetchAccessibleResources(httpClient, tr.AccessToken)
	if err != nil {
		fmt.Fprintf(stderr, "could not list accessible sites: %v\n", err)
		return exitNetErr, err
	}
	if len(resources) == 0 {
		fmt.Fprintln(stderr, "the authorized account has no accessible Atlassian sites")
		return exitAuthErr, fmt.Errorf("no accessible resources")
	}

	res, err := pickResource(resources, site, reader, stdout)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitErr, err
	}

	creds := stored
	creds.Mode = ModeOAuth
	creds.ClientID = clientID
	// The bundled app's secret is not persisted: it comes from the binary, so
	// rotating it only requires shipping a new release, not a re-login by
	// every user. Custom apps have nowhere else to keep theirs.
	if clientID == DefaultClientID {
		creds.ClientSecret = ""
	} else {
		creds.ClientSecret = clientSecret
	}
	creds.AccessToken = tr.AccessToken
	creds.RefreshToken = tr.RefreshToken
	creds.Expiry = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	creds.CloudID = res.ID
	creds.Site = subdomainFromURL(res.URL)
	if tr.Scope != "" {
		creds.Scopes = tr.Scope
	} else {
		creds.Scopes = scopes
	}

	if err := WriteCreds(creds); err != nil {
		fmt.Fprintf(stderr, "saving credentials: %v\n", err)
		return exitErr, err
	}

	path, _ := CredsPath()
	fmt.Fprintf(stdout, "logged in to %s (%s) via OAuth — tokens auto-refresh from now on\n", res.Name, res.URL)
	fmt.Fprintf(stdout, "credentials saved to %s (auth_mode=oauth)\n", path)
	return exitOK, nil
}

func printLoginHelp(w io.Writer) {
	fmt.Fprint(w, `login — OAuth 2.0 (3LO) browser login. Authorize in the browser and you're done.

USAGE:
  login [--site NAME|URL] [--no-browser] [--scopes "s1 s2"]
        [--client-id ID [--client-secret SECRET]] [--print-redirect-uri]

No app registration needed: released binaries carry the shared "Lybel Skills"
app. Tokens auto-refresh, so this is a one-time step.

  --site        pick the Atlassian site when your account has several
  --no-browser  print the URL instead of opening a browser (headless/SSH)
  --client-id   use your own Atlassian app instead of the built-in one
                (add --client-secret if that app is a confidential client)
`)
}

func readTrimmedLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// builtinClientSecret returns the secret bundled with DefaultClientID: the
// one baked in at release time, or the env override used by local builds.
func builtinClientSecret() string {
	if s := os.Getenv(clientSecretEnv); s != "" {
		return s
	}
	return DefaultClientSecret
}

// newPKCE returns a code_verifier and its S256 challenge (RFC 7636). Atlassian
// still requires the client secret, but PKCE binds the authorization code to
// this process, so an intercepted code is useless on its own.
func newPKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate code verifier: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

// waitForCallback serves the loopback redirect endpoint, opens the browser
// and returns the authorization code once Atlassian redirects back.
func waitForCallback(authURL, state string, port int, noBrowser bool, stdout, stderr io.Writer) (string, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return "", fmt.Errorf("cannot listen on port %d (is another login running?): %w", port, err)
	}

	type result struct {
		code string
		err  error
	}
	resultCh := make(chan result, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if e := q.Get("error"); e != "" {
			http.Error(w, "Authorization failed: "+e, http.StatusBadRequest)
			resultCh <- result{err: fmt.Errorf("authorization denied: %s (%s)", e, q.Get("error_description"))}
			return
		}
		if q.Get("state") != state {
			http.Error(w, "State mismatch — possible CSRF, try again.", http.StatusBadRequest)
			resultCh <- result{err: fmt.Errorf("state mismatch on callback")}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<html><body style=\"font-family:sans-serif\"><h3>Login complete ✔</h3><p>You can close this tab and return to the terminal.</p></body></html>")
		resultCh <- result{code: q.Get("code")}
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	fmt.Fprintln(stdout, "Open this URL to authorize:")
	fmt.Fprintln(stdout, "  "+authURL)
	if !noBrowser {
		if err := openBrowser(authURL); err != nil {
			fmt.Fprintf(stderr, "(could not open browser automatically: %v)\n", err)
		}
	}
	fmt.Fprintln(stdout, "Waiting for authorization…")

	select {
	case r := <-resultCh:
		if r.err != nil {
			return "", r.err
		}
		if r.code == "" {
			return "", fmt.Errorf("callback carried no authorization code")
		}
		return r.code, nil
	case <-time.After(loginTimeout):
		return "", fmt.Errorf("timed out waiting for authorization (%s)", loginTimeout)
	}
}

// openBrowser launches the platform default browser. On WSL it falls back to
// the Windows side (wslview, then explorer.exe).
func openBrowser(u string) error {
	var candidates [][]string
	switch runtime.GOOS {
	case "darwin":
		candidates = [][]string{{"open", u}}
	case "windows":
		candidates = [][]string{{"rundll32", "url.dll,FileProtocolHandler", u}}
	default:
		if isWSL() {
			candidates = [][]string{{"wslview", u}, {"explorer.exe", u}, {"xdg-open", u}}
		} else {
			candidates = [][]string{{"xdg-open", u}}
		}
	}
	var lastErr error
	for _, c := range candidates {
		if _, err := exec.LookPath(c[0]); err != nil {
			lastErr = err
			continue
		}
		if err := exec.Command(c[0], c[1:]...).Start(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no browser launcher found")
	}
	return lastErr
}

func isWSL() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	return err == nil && strings.Contains(strings.ToLower(string(data)), "microsoft")
}

// fetchAccessibleResources lists the sites the token can reach; this is where
// the cloudId for the API gateway comes from.
func fetchAccessibleResources(client Doer, accessToken string) ([]accessibleResource, error) {
	req, err := http.NewRequest(http.MethodGet, APIGateway+"/oauth/token/accessible-resources", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("accessible-resources returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var resources []accessibleResource
	if err := json.Unmarshal(body, &resources); err != nil {
		return nil, fmt.Errorf("parse accessible-resources: %w", err)
	}
	return resources, nil
}

// pickResource selects the target site: --site match, single-site shortcut,
// or interactive prompt.
func pickResource(resources []accessibleResource, site string, reader *bufio.Reader, stdout io.Writer) (accessibleResource, error) {
	if site != "" {
		needle := strings.ToLower(site)
		for _, r := range resources {
			if strings.Contains(strings.ToLower(r.URL), needle) || strings.EqualFold(r.Name, site) {
				return r, nil
			}
		}
		return accessibleResource{}, fmt.Errorf("no accessible site matches %q", site)
	}
	if len(resources) == 1 {
		return resources[0], nil
	}
	fmt.Fprintln(stdout, "Multiple sites accessible:")
	for i, r := range resources {
		fmt.Fprintf(stdout, "  %d) %s (%s)\n", i+1, r.Name, r.URL)
	}
	fmt.Fprint(stdout, "Choose [1]: ")
	line, err := readTrimmedLine(reader)
	if err != nil {
		return accessibleResource{}, err
	}
	idx := 1
	if line != "" {
		if _, err := fmt.Sscanf(line, "%d", &idx); err != nil || idx < 1 || idx > len(resources) {
			return accessibleResource{}, fmt.Errorf("invalid choice %q", line)
		}
	}
	return resources[idx-1], nil
}

func subdomainFromURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	return strings.TrimSuffix(host, ".atlassian.net")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
