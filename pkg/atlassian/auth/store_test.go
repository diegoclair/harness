package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// redirectConfigDir points os.UserConfigDir at a temp dir for the test.
func redirectConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("AppData", dir)
	case "darwin":
		t.Skip("cannot redirect os.UserConfigDir on darwin")
	default:
		t.Setenv("XDG_CONFIG_HOME", dir)
	}
	return dir
}

func TestWriteReadCreds_RoundTrip(t *testing.T) {
	redirectConfigDir(t)

	in := Credentials{
		Mode:         ModeOAuth,
		Email:        "d@example.com",
		Token:        "api-tok",
		ClientID:     "cid",
		ClientSecret: "csec",
		AccessToken:  "at",
		RefreshToken: "rt",
		Expiry:       time.Unix(1900000000, 0),
		CloudID:      "cloud-uuid",
		Site:         "lybel",
		Scopes:       "offline_access read:page:confluence",
	}
	if err := WriteCreds(in); err != nil {
		t.Fatalf("WriteCreds: %v", err)
	}

	out, err := ReadCreds()
	if err != nil {
		t.Fatalf("ReadCreds: %v", err)
	}
	if out != in {
		t.Errorf("round-trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

func TestWriteCreds_PreservesBothModes(t *testing.T) {
	redirectConfigDir(t)

	// Start with API-token creds (what setup writes).
	if err := WriteCreds(Credentials{Mode: ModeAPIToken, Email: "d@example.com", Token: "api-tok"}); err != nil {
		t.Fatal(err)
	}
	// Add an OAuth grant on top, as login does: read-modify-write.
	c, err := ReadCreds()
	if err != nil {
		t.Fatal(err)
	}
	c.Mode = ModeOAuth
	c.ClientID, c.ClientSecret, c.RefreshToken, c.CloudID = "cid", "csec", "rt", "cu"
	if err := WriteCreds(c); err != nil {
		t.Fatal(err)
	}

	out, _ := ReadCreds()
	if !out.HasAPIToken() || !out.HasOAuth() {
		t.Errorf("expected both modes preserved, got %+v", out)
	}
}

func TestWriteCreds_Permissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX perms not enforced on windows")
	}
	redirectConfigDir(t)

	if err := WriteCreds(Credentials{Email: "e", Token: "t"}); err != nil {
		t.Fatal(err)
	}
	path, _ := CredsPath()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("credentials perms = %o, want 0600", fi.Mode().Perm())
	}
}

func TestParseCreds_IgnoresUnknownAndComments(t *testing.T) {
	c := ParseCreds([]byte("# comment\nemail=a@b.c\nfuture_key=x\n\ntoken=tt\n"))
	if c.Email != "a@b.c" || c.Token != "tt" {
		t.Errorf("got %+v", c)
	}
}

func TestReadCreds_NotExist(t *testing.T) {
	redirectConfigDir(t)
	_, err := ReadCreds()
	if !os.IsNotExist(err) {
		t.Errorf("want IsNotExist, got %v", err)
	}
}

func TestAcquireLock_BlocksAndReleases(t *testing.T) {
	redirectConfigDir(t)

	release, err := acquireLock()
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	lp, _ := lockPath()
	if _, err := os.Stat(lp); err != nil {
		t.Fatalf("lock file missing: %v", err)
	}

	release()
	if _, err := os.Stat(lp); !os.IsNotExist(err) {
		t.Errorf("lock file not removed after release")
	}

	// Reacquire must work immediately.
	release2, err := acquireLock()
	if err != nil {
		t.Fatalf("second acquire: %v", err)
	}
	release2()
}

func TestAcquireLock_StaleLockIsBroken(t *testing.T) {
	dir := redirectConfigDir(t)

	lp := filepath.Join(dir, "atlassian", "credentials.lock")
	if err := os.MkdirAll(filepath.Dir(lp), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lp, []byte("999999\n"), 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * lockStaleAfter)
	if err := os.Chtimes(lp, old, old); err != nil {
		t.Fatal(err)
	}

	release, err := acquireLock()
	if err != nil {
		t.Fatalf("stale lock not broken: %v", err)
	}
	release()
}

func TestCredsFileFormat_SortedKeys(t *testing.T) {
	redirectConfigDir(t)
	if err := WriteCreds(Credentials{Email: "e", Token: "t", ClientID: "c"}); err != nil {
		t.Fatal(err)
	}
	path, _ := CredsPath()
	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for i := 1; i < len(lines); i++ {
		if lines[i-1] > lines[i] {
			t.Errorf("keys not sorted: %q before %q", lines[i-1], lines[i])
		}
	}
}
