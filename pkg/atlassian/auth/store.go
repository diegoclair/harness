// Package auth centralizes Atlassian authentication for every skill in this
// repo. Two modes are supported:
//
//   - apitoken: Basic auth with email + API token (id.atlassian.com tokens,
//     max lifetime 1 year since Dec/2024). Calls go to the site domain
//     (https://<cloud>.atlassian.net).
//   - oauth: OAuth 2.0 (3LO) with rotating refresh tokens, obtained by `login`
//     as a public client with PKCE — no client_secret, so the shared app id
//     ships in the binary and users authorize in the browser without
//     registering anything. Calls go through the API gateway
//     (https://api.atlassian.com/ex/<product>/<cloudId>).
//
// Credentials live in <UserConfigDir>/atlassian/credentials (0600), shared
// across all atlassian skills. The file is a flat key=value store; this
// package is its single writer so both modes can coexist in one file.
package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Mode identifies how requests are authenticated.
type Mode string

const (
	ModeAPIToken Mode = "apitoken"
	ModeOAuth    Mode = "oauth"
)

// Credentials is the full content of the shared credentials file. Fields for
// both modes coexist: writing OAuth fields never drops a stored API token and
// vice versa.
type Credentials struct {
	Mode Mode // which mode Resolve should prefer; empty means infer

	// API-token mode.
	Email string
	Token string

	// OAuth (3LO) mode.
	ClientID     string
	ClientSecret string // empty for the default public client (PKCE)
	AccessToken  string
	RefreshToken string
	Expiry       time.Time // access-token expiry
	CloudID      string    // from oauth/token/accessible-resources
	Site         string    // cloud subdomain (e.g. "lybel"), for web links
	Scopes       string    // space-separated scopes granted at login
}

// HasOAuth reports whether the stored OAuth grant is usable. ClientSecret is
// not required: the default app is a public client authenticated by PKCE.
func (c Credentials) HasOAuth() bool {
	return c.ClientID != "" && c.RefreshToken != "" && c.CloudID != ""
}

// HasAPIToken reports whether Basic-auth credentials are present.
func (c Credentials) HasAPIToken() bool {
	return c.Email != "" && c.Token != ""
}

// CredsPath returns the canonical shared credentials file path.
func CredsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}
	return filepath.Join(dir, "atlassian", "credentials"), nil
}

// credsKeys maps file keys to struct fields. Kept in one place so ReadCreds
// and WriteCreds cannot drift.
const (
	keyMode         = "auth_mode"
	keyEmail        = "email"
	keyToken        = "token"
	keyClientID     = "oauth_client_id"
	keyClientSecret = "oauth_client_secret"
	keyAccessToken  = "oauth_access_token"
	keyRefreshToken = "oauth_refresh_token"
	keyExpiry       = "oauth_expiry"
	keyCloudID      = "oauth_cloud_id"
	keySite         = "oauth_site"
	keyScopes       = "oauth_scopes"
)

// ReadCreds parses the canonical credentials file. A missing file returns
// zero-value Credentials and an os.IsNotExist error.
func ReadCreds() (Credentials, error) {
	path, err := CredsPath()
	if err != nil {
		return Credentials{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Credentials{}, err
	}
	return ParseCreds(data), nil
}

// ParseCreds parses key=value credential data. Unknown keys are ignored.
func ParseCreds(data []byte) Credentials {
	var c Credentials
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		switch key {
		case keyMode:
			c.Mode = Mode(val)
		case keyEmail:
			c.Email = val
		case keyToken:
			c.Token = val
		case keyClientID:
			c.ClientID = val
		case keyClientSecret:
			c.ClientSecret = val
		case keyAccessToken:
			c.AccessToken = val
		case keyRefreshToken:
			c.RefreshToken = val
		case keyExpiry:
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				c.Expiry = time.Unix(n, 0)
			}
		case keyCloudID:
			c.CloudID = val
		case keySite:
			c.Site = val
		case keyScopes:
			c.Scopes = val
		}
	}
	return c
}

// WriteCreds serializes credentials to the canonical path (0600), creating
// parent dirs. The write is atomic (temp file + rename) so a concurrent
// reader never sees a partial file.
func WriteCreds(c Credentials) error {
	path, err := CredsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	kv := map[string]string{}
	set := func(k, v string) {
		if v != "" {
			kv[k] = v
		}
	}
	set(keyMode, string(c.Mode))
	set(keyEmail, c.Email)
	set(keyToken, c.Token)
	set(keyClientID, c.ClientID)
	set(keyClientSecret, c.ClientSecret)
	set(keyAccessToken, c.AccessToken)
	set(keyRefreshToken, c.RefreshToken)
	if !c.Expiry.IsZero() {
		kv[keyExpiry] = strconv.FormatInt(c.Expiry.Unix(), 10)
	}
	set(keyCloudID, c.CloudID)
	set(keySite, c.Site)
	set(keyScopes, c.Scopes)

	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, kv[k])
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}

// lockPath is the sibling lock file used to serialize refresh-token rotation
// across concurrent CLI processes (Atlassian rotates the refresh token on
// every use — losing the newest one invalidates the grant).
func lockPath() (string, error) {
	path, err := CredsPath()
	if err != nil {
		return "", err
	}
	return path + ".lock", nil
}

const (
	lockRetryInterval = 100 * time.Millisecond
	lockTimeout       = 10 * time.Second
	lockStaleAfter    = 30 * time.Second
)

// acquireLock takes an exclusive advisory lock via O_CREATE|O_EXCL, which is
// atomic on every OS we ship to. Locks older than lockStaleAfter are treated
// as leftovers from a crashed process and removed.
func acquireLock() (release func(), err error) {
	path, err := lockPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	deadline := time.Now().Add(lockTimeout)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(path) }, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("acquire credentials lock: %w", err)
		}
		if fi, serr := os.Stat(path); serr == nil && time.Since(fi.ModTime()) > lockStaleAfter {
			os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("credentials file is locked by another process (%s); remove it if stale", path)
		}
		time.Sleep(lockRetryInterval)
	}
}
