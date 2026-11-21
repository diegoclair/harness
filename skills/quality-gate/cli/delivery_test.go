package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func deliveryRepo(t *testing.T, branch string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, configName, "ruleset: go\n")
	writeFile(t, root, "kept.go", "package x\n")
	writeFile(t, root, "gone.go", "package x\n")
	writeFile(t, root, "old.json", "{}\n")
	gitIn(t, root, "init", "-q", "-b", branch)
	gitIn(t, root, "add", "-A")
	commitIn(t, root, "base")
	return root
}

func commitIn(t *testing.T, root, message string) {
	t.Helper()
	gitIn(t, root, "-c", "user.name=probe", "-c", "user.email=probe@example.com", "-c", "commit.gpgsign=false",
		"-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", message)
}

func runDelivery(t *testing.T, root string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut strings.Builder
	code := cmdDelivery(append([]string{"--dir", root}, args...), &out, &errOut)
	return code, out.String(), errOut.String()
}

// A tool scoping its run on the delivery misses whatever the list leaves out, so a deletion, both sides of a
// committed rename, a staged edit and each file of a new folder are all listed.
func TestDeliveryListsEveryKindOfChange(t *testing.T) {
	root := deliveryRepo(t, "main")
	gitIn(t, root, "checkout", "-q", "-b", "work")
	gitIn(t, root, "mv", "old.json", "renamed.json")
	commitIn(t, root, "rename")
	if err := os.Remove(filepath.Join(root, "gone.go")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "kept.go", "package x\n\nvar v = 1\n")
	gitIn(t, root, "add", "kept.go")
	if err := os.MkdirAll(filepath.Join(root, "fresh", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "fresh/deep/a.go", "package deep\n")

	code, out, errOut := runDelivery(t, root, "--format", "json")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	var d Delivery
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		t.Fatal(err)
	}
	want := []string{"fresh/deep/a.go", "gone.go", "kept.go", "old.json", "renamed.json"}
	if strings.Join(d.Paths, ",") != strings.Join(want, ",") {
		t.Fatalf("paths %v, want %v", d.Paths, want)
	}
	if d.Base == "" || d.Head == "" || d.Base == d.Head {
		t.Fatalf("base %q head %q: the merge-base and HEAD differ on a branch with a commit of its own", d.Base, d.Head)
	}
}

// A partial list would read as "nothing else changed", so a delivery git cannot read whole is an error.
func TestDeliveryRefusesWhatItCannotReadWhole(t *testing.T) {
	t.Run("no base branch", func(t *testing.T) {
		root := deliveryRepo(t, "trunk")
		if code, _, errOut := runDelivery(t, root); code != exitUsage || !strings.Contains(errOut, "no base branch") {
			t.Fatalf("exit %d: %s", code, errOut)
		}
	})
	t.Run("base git cannot relate", func(t *testing.T) {
		root := deliveryRepo(t, "main")
		if code, _, errOut := runDelivery(t, root, "--since", "no-such-ref"); code != exitUsage || !strings.Contains(errOut, "merge-base") {
			t.Fatalf("exit %d: %s", code, errOut)
		}
	})
	t.Run("not a repository", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, configName, "ruleset: go\n")
		if code, _, errOut := runDelivery(t, root, "--since", "main"); code != exitUsage {
			t.Fatalf("exit %d: %s", code, errOut)
		}
	})
}

// git status collapses a new folder into one line by default; the gate must still read the files inside it.
func TestCheckScansAFileInANewFolder(t *testing.T) {
	root := deliveryRepo(t, "main")
	if err := os.MkdirAll(filepath.Join(root, "fresh"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "fresh/a.go", "package fresh\n")
	cfg, err := loadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	files, err := changedFiles(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(files, ",") != "fresh/a.go" {
		t.Fatalf("changed %v, want only the new folder's file", files)
	}
}
