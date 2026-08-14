package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// TokenURL is Atlassian's OAuth token endpoint. Overridable in tests.
var TokenURL = "https://auth.atlassian.com/oauth/token"

// APIGateway is the base of Atlassian's OAuth API gateway. Overridable in tests.
var APIGateway = "https://api.atlassian.com"

// Doer is the minimal HTTP client interface, injectable in tests.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Authorizer signs requests and knows which base URL each product must use —
// the site domain for Basic auth, the api.atlassian.com gateway for OAuth.
type Authorizer interface {
	// Apply sets the Authorization header, refreshing tokens if needed.
	Apply(req *http.Request) error
	// ConfluenceBase returns the Confluence API root, ending in /wiki.
	ConfluenceBase() string
	// JiraBase returns the Jira product root (prepend to /rest/api/3, …).
	JiraBase() string
	// ConfluenceWebBase returns the site root for browser links, ending in
	// /wiki. It differs from ConfluenceBase under OAuth, where API calls go
	// through the gateway but humans need the site domain.
	ConfluenceWebBase() string
	// Kind reports the active mode, for diagnostics.
	Kind() Mode
}

// ---------- Basic (API token) ----------

// Basic authenticates with email + API token against the site domain.
type Basic struct {
	Email string
	Token string
	Cloud string // subdomain, e.g. "lybel"
}

func (b Basic) Apply(req *http.Request) error {
	raw := b.Email + ":" + b.Token
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(raw)))
	return nil
}

func (b Basic) ConfluenceBase() string {
	return fmt.Sprintf("https://%s.atlassian.net/wiki", b.Cloud)
}

func (b Basic) JiraBase() string {
	return fmt.Sprintf("https://%s.atlassian.net", b.Cloud)
}

// ConfluenceWebBase is the same as ConfluenceBase for Basic auth: calls
// already go to the site domain.
func (b Basic) ConfluenceWebBase() string { return b.ConfluenceBase() }

func (b Basic) Kind() Mode { return ModeAPIToken }

// ---------- OAuth (3LO) ----------

// OAuth authenticates with a Bearer token from a 3LO grant, refreshing it
// transparently. Refresh persists the rotated refresh token under a file
// lock, and re-reads the store first so concurrent processes reuse a token
// refreshed by a sibling instead of burning the rotation chain.
type OAuth struct {
	mu    sync.Mutex
	creds Credentials

	// HTTP is used for token refresh. Defaults to a 30s-timeout client.
	HTTP Doer
	// Now is the clock, injectable in tests.
	Now func() time.Time
	// Persist disables writing refreshed tokens back to disk when false
	// (tests). Defaults to true via NewOAuth.
	Persist bool
}

// NewOAuth builds an OAuth authorizer from stored credentials.
func NewOAuth(c Credentials) *OAuth {
	return &OAuth{
		creds:   c,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		Now:     time.Now,
		Persist: true,
	}
}

func (o *OAuth) Apply(req *http.Request) error {
	tok, err := o.accessToken()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	return nil
}

func (o *OAuth) ConfluenceBase() string {
	return fmt.Sprintf("%s/ex/confluence/%s/wiki", APIGateway, o.creds.CloudID)
}

func (o *OAuth) JiraBase() string {
	return fmt.Sprintf("%s/ex/jira/%s", APIGateway, o.creds.CloudID)
}

// ConfluenceWebBase returns the site domain so printed links are clickable —
// the gateway URL used for API calls is not. Falls back to the API base when
// the site subdomain is unknown (pre-0.15 grants).
func (o *OAuth) ConfluenceWebBase() string {
	if o.creds.Site == "" {
		return o.ConfluenceBase()
	}
	return fmt.Sprintf("https://%s.atlassian.net/wiki", o.creds.Site)
}

func (o *OAuth) Kind() Mode { return ModeOAuth }

// Creds returns a copy of the current credentials (post-refresh state).
func (o *OAuth) Creds() Credentials {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.creds
}

// expirySkew forces a refresh slightly before the real expiry so a token
// never dies mid-request.
const expirySkew = 60 * time.Second

// clientSecret resolves the secret used to refresh. Grants on the bundled app
// take it from the binary, so a rotated secret reaches users through a normal
// release instead of forcing everyone to log in again. Custom apps keep
// theirs in the credentials file.
func (o *OAuth) clientSecret() string {
	if o.creds.ClientSecret != "" {
		return o.creds.ClientSecret
	}
	if o.creds.ClientID == DefaultClientID {
		return builtinClientSecret()
	}
	return ""
}

func (o *OAuth) accessToken() (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.creds.AccessToken != "" && o.Now().Add(expirySkew).Before(o.creds.Expiry) {
		return o.creds.AccessToken, nil
	}
	if o.creds.RefreshToken == "" {
		return "", fmt.Errorf("no OAuth refresh token stored — run `login` again")
	}

	release, err := acquireLock()
	if err != nil {
		return "", err
	}
	defer release()

	// Another process may have refreshed while we waited on the lock.
	if o.Persist {
		if stored, rerr := ReadCreds(); rerr == nil && stored.AccessToken != "" &&
			o.Now().Add(expirySkew).Before(stored.Expiry) {
			o.creds.AccessToken = stored.AccessToken
			o.creds.RefreshToken = stored.RefreshToken
			o.creds.Expiry = stored.Expiry
			return o.creds.AccessToken, nil
		}
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {o.creds.ClientID},
		"refresh_token": {o.creds.RefreshToken},
	}
	if secret := o.clientSecret(); secret != "" {
		form.Set("client_secret", secret)
	}
	tr, err := requestToken(o.HTTP, form)
	if err != nil {
		if isInvalidGrant(err) {
			return "", fmt.Errorf("OAuth refresh token expired or revoked — run `login` again (%w)", err)
		}
		return "", err
	}

	o.creds.AccessToken = tr.AccessToken
	if tr.RefreshToken != "" {
		o.creds.RefreshToken = tr.RefreshToken // rotation: always keep the newest
	}
	o.creds.Expiry = o.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)

	if o.Persist {
		if err := WriteCreds(o.creds); err != nil {
			return "", fmt.Errorf("persist refreshed token: %w", err)
		}
	}
	return o.creds.AccessToken, nil
}

// tokenResponse is the OAuth token endpoint response.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
}

// tokenError carries the OAuth error envelope for classification.
type tokenError struct {
	StatusCode int
	Code       string
	Desc       string
}

func (e *tokenError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("token endpoint %d: %s (%s)", e.StatusCode, e.Code, e.Desc)
	}
	return fmt.Sprintf("token endpoint returned %d", e.StatusCode)
}

func isInvalidGrant(err error) bool {
	var te *tokenError
	return errors.As(err, &te) && (te.Code == "invalid_grant" || te.StatusCode == http.StatusForbidden)
}

// IsReloginRequired reports whether err (anywhere in its chain) means the
// OAuth grant is dead — expired/revoked refresh token — so the only fix is
// running `login` again. Lets callers map it to an invalid-auth exit code
// instead of a network error.
func IsReloginRequired(err error) bool {
	return isInvalidGrant(err)
}

// requestToken POSTs to the token endpoint and parses the response.
func requestToken(client Doer, form url.Values) (tokenResponse, error) {
	req, err := http.NewRequest(http.MethodPost, TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return tokenResponse{}, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tokenResponse{}, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var envelope struct {
			Error string `json:"error"`
			Desc  string `json:"error_description"`
		}
		_ = json.Unmarshal(body, &envelope)
		return tokenResponse{}, &tokenError{StatusCode: resp.StatusCode, Code: envelope.Error, Desc: envelope.Desc}
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return tokenResponse{}, fmt.Errorf("parse token response: %w", err)
	}
	if tr.AccessToken == "" {
		return tokenResponse{}, fmt.Errorf("token response missing access_token")
	}
	return tr, nil
}

// ---------- Resolution ----------

// Options configures Resolve.
type Options struct {
	// Email/Token from CLI flags (highest priority, forces Basic).
	Email string
	Token string
	// Cloud is the subdomain for Basic mode (already resolved by the caller
	// from flags/env/config).
	Cloud string
	// Stderr receives migration warnings. Defaults to os.Stderr.
	Stderr io.Writer
	// LegacyCredsPaths are optional per-skill fallback files (email/token
	// only) tried when the canonical file is absent.
	LegacyCredsPaths []string
}

// Resolve picks the effective Authorizer:
//
//  1. explicit flags (Basic)
//  2. $ATLASSIAN_EMAIL + $ATLASSIAN_API_TOKEN (Basic)
//  3. stored OAuth grant (unless auth_mode forces apitoken)
//  4. stored email+token, including legacy per-skill fallback paths (Basic)
//
// Basic modes require Options.Cloud; OAuth does not (it uses the cloudId).
func Resolve(opts Options) (Authorizer, error) {
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	if opts.Email != "" && opts.Token != "" {
		return basicWithCloud(opts.Email, opts.Token, opts.Cloud)
	}
	if e, t := os.Getenv("ATLASSIAN_EMAIL"), os.Getenv("ATLASSIAN_API_TOKEN"); e != "" && t != "" {
		return basicWithCloud(e, t, opts.Cloud)
	}

	creds, err := ReadCreds()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if creds.HasOAuth() && creds.Mode != ModeAPIToken {
		return NewOAuth(creds), nil
	}
	if creds.HasAPIToken() {
		return basicWithCloud(creds.Email, creds.Token, opts.Cloud)
	}

	// Legacy per-skill files predate OAuth and hold email/token only.
	for _, p := range opts.LegacyCredsPaths {
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			continue
		}
		legacy := ParseCreds(data)
		if legacy.HasAPIToken() {
			canonical, _ := CredsPath()
			fmt.Fprintf(stderr,
				"warning: credentials found at legacy path %s — run `setup` to migrate to %s\n", p, canonical)
			return basicWithCloud(legacy.Email, legacy.Token, opts.Cloud)
		}
	}

	return nil, fmt.Errorf(
		"no Atlassian credentials found.\n" +
			"Options (use any one):\n" +
			"  1. `login`  — OAuth with your own app (auto-refreshing, recommended)\n" +
			"  2. `setup`  — email + API token (expires after at most 1 year)\n" +
			"  3. Env:     ATLASSIAN_EMAIL=you@example.com ATLASSIAN_API_TOKEN=<api-token>")
}

func basicWithCloud(email, token, cloud string) (Authorizer, error) {
	if cloud == "" {
		return nil, fmt.Errorf("no Atlassian cloud subdomain configured — run `setup`, or export ATLASSIAN_CLOUD=mycompany")
	}
	return Basic{Email: email, Token: token, Cloud: cloud}, nil
}
