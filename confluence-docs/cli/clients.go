// clients.go — shared construction of the Confluence API client.
package main

import (
	"os"
	"path/filepath"

	"github.com/diegoclair/skills/pkg/atlassian/adf"
	"github.com/diegoclair/skills/pkg/atlassian/auth"
)

// newClient resolves the effective authorizer (flags → env → stored OAuth
// grant → stored/legacy email+token) and returns a ready-to-use client.
// Cloud is only required for Basic auth; OAuth routes through the
// api.atlassian.com gateway, so auth.Resolve errors on a missing cloud only
// when it actually needs one.
func newClient(cloudFlag, emailFlag, tokenFlag string) (*adf.ConfluenceClient, error) {
	a, err := auth.Resolve(auth.Options{
		Email:            emailFlag,
		Token:            tokenFlag,
		Cloud:            adf.ResolveCloud(cloudFlag),
		LegacyCredsPaths: legacyCredsPaths(),
	})
	if err != nil {
		return nil, err
	}
	return adf.NewClientWithAuthorizer(a), nil
}

// legacyCredsPaths lists the pre-shared-store per-skill credential files,
// tried by auth.Resolve as a last resort (with a migration warning).
func legacyCredsPaths() []string {
	var paths []string
	if dir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(dir, "confluence-docs", "credentials"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "confluence-docs", "credentials"))
	}
	return paths
}
