package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/diegoclair/skills/pkg/atlassian/auth"
	"github.com/diegoclair/skills/pkg/atlassian/jira"
	"github.com/diegoclair/skills/pkg/atlassian/setup"
)

// parseCommonFlags consumes the cross-command flags (--cloud / --email /
// --token) before the per-command flag loop, mirroring the same pattern
// confluence-docs uses. Returns the remaining args and the resolved values.
func parseCommonFlags(args []string) (remaining []string, cloud, email, token string, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--cloud":
			if i+1 >= len(args) {
				return nil, "", "", "", fmt.Errorf("flag --cloud requires a value")
			}
			cloud = args[i+1]
			i++
		case "--email":
			if i+1 >= len(args) {
				return nil, "", "", "", fmt.Errorf("flag --email requires a value")
			}
			email = args[i+1]
			i++
		case "--token":
			if i+1 >= len(args) {
				return nil, "", "", "", fmt.Errorf("flag --token requires a value")
			}
			token = args[i+1]
			i++
		default:
			remaining = append(remaining, a)
		}
	}
	return remaining, cloud, email, token, nil
}

// buildClient resolves credentials via the shared auth package and returns a
// Jira client. Returns (nil, false) on failure, after printing a helpful
// error to stderr.
//
// Resolution order (handled by auth.Resolve):
//  1. Explicit flags (--email / --token, Basic)
//  2. Environment variables (ATLASSIAN_EMAIL / ATLASSIAN_API_TOKEN, Basic)
//  3. Stored OAuth grant (from `jira-tickets login`)
//  4. Stored email+token, including legacy per-skill paths (Basic)
//
// The cloud subdomain (flag → ATLASSIAN_CLOUD → per-skill config) is only
// required in Basic mode; OAuth routes by cloudId.
func buildClient(cloud, email, token string, stderr io.Writer) (*jira.Client, bool) {
	a, err := auth.Resolve(auth.Options{
		Email:            email,
		Token:            token,
		Cloud:            resolveCloud(cloud),
		Stderr:           stderr,
		LegacyCredsPaths: legacyCredsPaths(),
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return nil, false
	}
	return jira.NewClientWithAuthorizer(a), true
}

// legacyCredsPaths lists the pre-OAuth per-skill credential files, tried by
// auth.Resolve when the canonical atlassian-wide file is absent.
func legacyCredsPaths() []string {
	var paths []string
	if dir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(dir, "jira-tickets", "credentials"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "jira-tickets", "credentials"))
	}
	return paths
}

// resolveCloud picks the cloud subdomain from (in order) explicit flag,
// ATLASSIAN_CLOUD env var, or the per-skill config file written by `setup`.
func resolveCloud(flag string) string {
	if flag != "" {
		return flag
	}
	if env := os.Getenv("ATLASSIAN_CLOUD"); env != "" {
		return env
	}
	cfg := setup.ReadConfigFile()
	return cfg.Cloud
}

// parseStringList splits a comma-separated list, trimming whitespace and
// dropping empty entries. Used for --fields / --labels / etc.
func parseStringList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// issueWebURL builds the Jira UI URL for an issue, e.g.
// https://mycompany.atlassian.net/browse/PROJ-123. Returns "" when the site
// cannot be determined.
func issueWebURL(client *jira.Client, key string) string {
	if client == nil || client.Auth == nil || key == "" {
		return ""
	}
	// Basic mode: JiraBase is the browsable site domain.
	if client.Auth.Kind() == auth.ModeAPIToken {
		return client.Auth.JiraBase() + "/browse/" + key
	}
	// OAuth mode: JiraBase is the API gateway; use the stored site subdomain.
	if o, ok := client.Auth.(*auth.OAuth); ok {
		if site := o.Creds().Site; site != "" {
			return fmt.Sprintf("https://%s.atlassian.net/browse/%s", site, key)
		}
	}
	return ""
}
