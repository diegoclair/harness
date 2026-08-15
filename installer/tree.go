package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// markerFile records that the harness owns a skill directory. Installing
// replaces the directory wholesale, so it refuses to wipe anything it did not
// put there.
const markerFile = ".harness-installed"

// treeProvider yields the repo tree an install copies from.
type treeProvider interface {
	// root returns a local directory holding the repo tree (skills/, agents/).
	root() (string, error)
	describe() string
}

// localTree is a clone already on disk (`--from PATH`).
type localTree struct{ path string }

func (l localTree) root() (string, error) {
	if _, err := os.Stat(filepath.Join(l.path, "skills")); err != nil {
		return "", fmt.Errorf("%s does not look like a harness clone: no skills/ directory", l.path)
	}
	return l.path, nil
}
func (l localTree) describe() string { return l.path }

// remoteTree downloads the repo tarball once and reuses it for every artifact
// in the same run.
type remoteTree struct {
	repo string
	ref  string
	tmp  string

	extracted string
	err       error
}

func (r *remoteTree) describe() string { return r.repo + "@" + r.ref }

func (r *remoteTree) root() (string, error) {
	if r.extracted != "" {
		return r.extracted, nil
	}
	if r.err != nil {
		return "", r.err
	}
	root, err := r.fetch()
	if err != nil {
		r.err = err
		return "", err
	}
	r.extracted = root
	return root, nil
}

func (r *remoteTree) fetch() (string, error) {
	url := fmt.Sprintf("https://codeload.github.com/%s/tar.gz/%s", r.repo, r.ref)
	archive := filepath.Join(r.tmp, "tree.tar.gz")
	if err := download(url, archive); err != nil {
		return "", fmt.Errorf("%w\n  check that %s exists and ref %q is valid", err, r.repo, r.ref)
	}

	dest := filepath.Join(r.tmp, "tree")
	if err := untar(archive, dest); err != nil {
		return "", err
	}
	// codeload wraps everything in a single <repo>-<ref> directory, and mangles
	// the ref (a leading "v" is dropped). Read the real name instead of guessing.
	entries, err := os.ReadDir(dest)
	if err != nil {
		return "", err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) != 1 {
		return "", fmt.Errorf("unexpected archive layout: %d top-level directories, want 1", len(dirs))
	}
	return filepath.Join(dest, dirs[0]), nil
}

// untar extracts a gzipped tarball, rejecting entries that would escape the
// destination (tar-slip) and skipping anything that is not a regular file or
// directory — a symlink in an untrusted archive can redirect a later write.
func untar(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("read archive: %w", err)
	}
	defer gz.Close()

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	cleanDest := filepath.Clean(dest)

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}

		target := filepath.Join(dest, filepath.FromSlash(hdr.Name))
		if target != cleanDest && !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry escapes destination: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := writeTarEntry(tr, target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		}
	}
}

func writeTarEntry(tr *tar.Reader, target string, mode os.FileMode) error {
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm()|0o200)
	if err != nil {
		return fmt.Errorf("create %s: %w", target, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, tr); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	return nil
}

func installAgent(a Artifact, root, ref string, out io.Writer) error {
	fmt.Fprintf(out, "Installing %s (agent) from %s\n", a.Name, ref)

	src := filepath.Join(root, "agents", a.Name+".md")
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("agent %s not found in the repo tree", a.Name)
	}
	dir, err := agentDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	dst := filepath.Join(dir, a.Name+".md")
	if err := backupIfForeign(src, dst, out); err != nil {
		return err
	}
	if err := copyFile(src, dst); err != nil {
		return err
	}
	fmt.Fprintf(out, "  Installed: %s\n", dst)
	return nil
}

// backupIfForeign preserves a hand-written agent before replacing it. Agents
// are single files with nowhere to keep a marker, and a skill's dependency can
// pull one in without the user ever naming it.
func backupIfForeign(src, dst string, out io.Writer) error {
	existing, err := os.ReadFile(dst)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	incoming, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if string(existing) == string(incoming) {
		return nil
	}
	backup := dst + ".bak"
	if err := os.WriteFile(backup, existing, 0o644); err != nil {
		return fmt.Errorf("back up %s: %w", dst, err)
	}
	fmt.Fprintf(out, "  Replacing a different %s; previous version saved to %s\n",
		filepath.Base(dst), backup)
	return nil
}

// clearSkillDir empties a skill directory before a fresh copy, refusing to
// touch one the harness did not install. bin/ survives only while the incoming
// payload still carries a binary; a skill that stopped shipping one would
// otherwise leave a stale executable on PATH forever.
func clearSkillDir(dir string, keepBin bool, name string, out io.Writer) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0o755)
	}
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	if _, err := os.Stat(filepath.Join(dir, markerFile)); err != nil {
		if err := adoptOrPreserve(dir, name, out); err != nil {
			return err
		}
	}
	for _, e := range entries {
		if e.Name() == "bin" {
			if keepBin {
				continue
			}
			if err := removeStaleBinary(dir, name, out); err != nil {
				return err
			}
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("clear %s: %w", dir, err)
		}
	}
	return nil
}

// adoptOrPreserve decides what to do with a skill directory carrying no marker.
// A bin/<name> is an installer's signature — nobody hand-writes one — so such a
// directory is a previous install, including one from the predecessor repo that
// never wrote markers, and is adopted. Anything else may be the user's own work
// and is moved aside rather than destroyed.
func adoptOrPreserve(dir, name string, out io.Writer) error {
	if _, err := os.Stat(filepath.Join(dir, "bin", binaryName(name))); err == nil {
		fmt.Fprintf(out, "  Adopting an existing install of %s\n", name)
		return nil
	}

	backup := dir + ".bak"
	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("clear %s: %w", backup, err)
	}
	if err := os.Rename(dir, backup); err != nil {
		return fmt.Errorf("%s already exists and was not installed by harness, "+
			"and could not be moved aside: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	fmt.Fprintf(out, "  %s already existed and was not installed by harness; "+
		"kept the previous contents in %s\n", dir, backup)
	return nil
}

// removeStaleBinary drops a binary the skill no longer ships, along with the
// PATH symlink pointing at it — leaving either behind keeps the old executable
// runnable indefinitely.
func removeStaleBinary(dir, name string, out io.Writer) error {
	if err := os.RemoveAll(filepath.Join(dir, "bin")); err != nil {
		return fmt.Errorf("remove stale bin/: %w", err)
	}
	fmt.Fprintf(out, "  %s no longer ships a binary; removed its bin/\n", name)

	userBin, err := userBinDir()
	if err != nil {
		return nil
	}
	link := filepath.Join(userBin, name)
	target, err := os.Readlink(link)
	if err != nil {
		return nil
	}
	if !strings.HasPrefix(target, dir+string(os.PathSeparator)) {
		return nil
	}
	if err := os.Remove(link); err != nil {
		return nil
	}
	fmt.Fprintf(out, "  Removed the stale PATH link %s\n", link)
	return nil
}

func copyTree(src, dst string) (int, error) {
	return copyTreeExcept(src, dst)
}

// copyTreeExcept copies a directory, skipping the named top-level entries.
func copyTreeExcept(src, dst string, skip ...string) (int, error) {
	skipped := map[string]bool{}
	for _, s := range skip {
		skipped[s] = true
	}

	count := 0
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if top, _, _ := strings.Cut(rel, string(os.PathSeparator)); skipped[top] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if err := copyFile(path, target); err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if err := os.Chmod(target, info.Mode().Perm()); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}
