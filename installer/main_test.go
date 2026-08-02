package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSelectSkills(t *testing.T) {
	tests := []struct {
		name    string
		names   []string
		all     bool
		want    []string
		wantErr string // substring expected on stderr; empty means success
	}{
		{name: "single", names: []string{"jira-tickets"}, want: []string{"jira-tickets"}},
		{
			name:  "several keep request order",
			names: []string{"jira-tickets", "confluence-docs"},
			want:  []string{"jira-tickets", "confluence-docs"},
		},
		{
			name:  "duplicates collapse",
			names: []string{"jira-tickets", "jira-tickets"},
			want:  []string{"jira-tickets"},
		},
		{name: "all", all: true, want: skillNames()},
		{name: "unknown name", names: []string{"confluence"}, wantErr: `unknown skill "confluence"`},
		{name: "nothing requested", wantErr: "no skill given"},
		{name: "all plus a name", all: true, names: []string{"jira-tickets"}, wantErr: "--all takes no skill names"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			got, code := selectSkills(tc.names, tc.all, &stderr)

			if tc.wantErr != "" {
				if code == exitOK {
					t.Fatalf("want failure, got exit 0 with %v", got)
				}
				if !strings.Contains(stderr.String(), tc.wantErr) {
					t.Errorf("stderr = %q, want it to mention %q", stderr.String(), tc.wantErr)
				}
				return
			}

			if code != exitOK {
				t.Fatalf("exit %d, stderr: %s", code, stderr.String())
			}
			var names []string
			for _, s := range got {
				names = append(names, s.Name)
			}
			if strings.Join(names, ",") != strings.Join(tc.want, ",") {
				t.Errorf("selected %v, want %v", names, tc.want)
			}
		})
	}
}

// A typo must be rejected before anything is downloaded, so a multi-skill run
// never leaves a partially applied selection.
func TestRunInstall_RejectsUnknownBeforeInstalling(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runInstall([]string{"confluence-docs", "nope"}, &stdout, &stderr)

	if code != exitInputErr {
		t.Fatalf("exit = %d, want %d", code, exitInputErr)
	}
	if strings.Contains(stdout.String(), "Installing") {
		t.Errorf("nothing should have been installed, stdout: %q", stdout.String())
	}
}

// A pinned tag belongs to one skill's release series.
func TestRunInstall_VersionWithMultipleSkills(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runInstall([]string{"--version", "jira-v0.4.1", "confluence-docs", "jira-tickets"}, &stdout, &stderr)

	if code != exitInputErr {
		t.Fatalf("exit = %d, want %d", code, exitInputErr)
	}
	if !strings.Contains(stderr.String(), "single skill") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestList(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"list"}, &out, &out); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	for _, s := range catalog {
		if !strings.Contains(out.String(), s.Name) {
			t.Errorf("%q missing from list output", s.Name)
		}
		if !strings.Contains(out.String(), s.Summary) {
			t.Errorf("summary for %q missing; the list is what an agent reads to offer choices", s.Name)
		}
	}
}

func TestUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"instal"}, &stdout, &stderr); code != exitInputErr {
		t.Fatalf("exit = %d, want %d", code, exitInputErr)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

// Every catalog entry must be installable: a missing tag prefix silently
// resolves to the wrong release series.
func TestCatalogIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range catalog {
		if s.Name == "" || s.TagPrefix == "" || s.Summary == "" {
			t.Errorf("incomplete catalog entry: %+v", s)
		}
		if !strings.HasSuffix(s.TagPrefix, "-v") {
			t.Errorf("%s: tag prefix %q should end in -v", s.Name, s.TagPrefix)
		}
		if seen[s.TagPrefix] {
			t.Errorf("%s: tag prefix %q is used by another skill", s.Name, s.TagPrefix)
		}
		seen[s.TagPrefix] = true
	}
}
