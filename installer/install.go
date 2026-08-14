package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/diegoclair/harness/pkg/release"
)

const defaultRepo = "diegoclair/harness"

// httpTimeout bounds each network call; a release archive is a few MB.
var httpClient = &http.Client{Timeout: 2 * time.Minute}

// installOptions carries what the caller decided, so install() itself makes no
// policy choices.
type installOptions struct {
	Repo string
	// Version pins a release tag. Empty resolves the newest for the prefix.
	Version string
	Out     io.Writer
}

// install runs the full pipeline for one skill: resolve version → download →
// extract → atomic binary install → skill payload → PATH → verify → hooks.
func installFromRelease(s Artifact, opts installOptions) error {
	out := opts.Out
	plat, err := platform()
	if err != nil {
		return err
	}

	version, err := resolveVersion(s, opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Installing %s %s for %s\n", s.Name, version, plat)

	dir, err := skillDir(s.Name)
	if err != nil {
		return err
	}
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", binDir, err)
	}

	archive := fmt.Sprintf("%s-%s.zip", s.Name, plat)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", opts.Repo, version, archive)

	tmp, err := os.MkdirTemp("", "skills-install-")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	fmt.Fprintf(out, "  Downloading %s\n", url)
	archivePath := filepath.Join(tmp, archive)
	if err := download(url, archivePath); err != nil {
		return err
	}

	fmt.Fprintln(out, "  Extracting...")
	extractDir := filepath.Join(tmp, "extracted")
	if err := unzip(archivePath, extractDir); err != nil {
		return err
	}

	installed, err := installPayload(s, extractDir, dir, binDir)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "  Installed binary + %d skill file(s) from archive.\n", installed)

	binPath := filepath.Join(binDir, binaryName(s.Name))
	if err := linkOnPath(binPath, s.Name, opts.Repo, out); err != nil {
		// Never fatal: the binary is installed and usable by absolute path.
		fmt.Fprintf(out, "  warning: %v\n", err)
	}

	return verify(s, binPath, version, dir, out)
}

// resolveVersion honours an explicit pin, then the skill's legacy env var,
// then the newest release carrying the skill's tag prefix. The repo tags every
// skill separately, so GitHub's "latest" pointer cannot be used.
func resolveVersion(s Artifact, opts installOptions) (string, error) {
	if opts.Version != "" {
		return opts.Version, nil
	}
	if v := os.Getenv("SKILL_VERSION"); v != "" {
		return v, nil
	}
	if s.VersionEnv != "" {
		if v := os.Getenv(s.VersionEnv); v != "" {
			return v, nil
		}
	}
	tag, err := release.FindLatestByPrefix(opts.Repo, s.TagPrefix, httpClient)
	if err != nil {
		return "", fmt.Errorf("resolving latest %s release: %w\n"+
			"  pin one explicitly with --version %s<X.Y.Z> if this persists", s.Name, err, s.TagPrefix)
	}
	return tag, nil
}

func download(url, dest string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}
	return nil
}

// unzip extracts an archive, rejecting entries that would escape the
// destination directory (zip-slip).
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(dest, filepath.FromSlash(f.Name))
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry escapes destination: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := writeZipEntry(f, target); err != nil {
			return err
		}
	}
	return nil
}

func writeZipEntry(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("read %s from archive: %w", f.Name, err)
	}
	defer rc.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode()|0o200)
	if err != nil {
		return fmt.Errorf("create %s: %w", target, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, rc); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	return nil
}

// installPayload places the binary and the skill files, returning how many
// files were written.
func installPayload(s Artifact, extractDir, dir, binDir string) (int, error) {
	bin := binaryName(s.Name)
	src := filepath.Join(extractDir, "bin", bin)
	if _, err := os.Stat(src); err != nil {
		return 0, fmt.Errorf("binary %s not found in archive", bin)
	}
	if err := installBinary(src, filepath.Join(binDir, bin)); err != nil {
		return 0, err
	}

	count := 0
	if err := copyFile(filepath.Join(extractDir, "SKILL.md"), filepath.Join(dir, "SKILL.md")); err == nil {
		count++
	}

	// Clean slate: a renamed reference file would otherwise linger forever.
	refSrc := filepath.Join(extractDir, "reference")
	refDst := filepath.Join(dir, "reference")
	entries, err := os.ReadDir(refSrc)
	if err != nil {
		return count, nil
	}
	if err := os.RemoveAll(refDst); err != nil {
		return count, fmt.Errorf("clear %s: %w", refDst, err)
	}
	if err := os.MkdirAll(refDst, 0o755); err != nil {
		return count, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if err := copyFile(filepath.Join(refSrc, e.Name()), filepath.Join(refDst, e.Name())); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// installBinary writes through a sibling temp file and renames. Copying onto a
// running executable fails with ETXTBSY, which is exactly the `<skill> update`
// path: the running binary asks the installer to replace it.
func installBinary(src, dst string) error {
	tmp := filepath.Join(filepath.Dir(dst), "."+filepath.Base(dst)+".new")
	if err := copyFile(src, tmp); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return fmt.Errorf("chmod %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("install binary: %w", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

// verify runs the installed binary and reports credential state. Unlike the
// shell installer it surfaces the skill's own diagnosis instead of collapsing
// every non-zero exit into "not configured" — a valid grant on a site missing
// the product used to be reported as missing credentials.
func verify(s Artifact, binPath, version, dir string, out io.Writer) error {
	fmt.Fprintln(out, "\nVerifying installation...")
	ver, err := exec.Command(binPath, "--version").Output()
	if err != nil {
		return fmt.Errorf("binary verification failed; %s may be corrupted: %w", binPath, err)
	}
	fmt.Fprintf(out, "  %s", ver)

	runPostInstall(binPath, out)

	fmt.Fprintln(out, "\nChecking credentials...")
	// Output(), not Run(): only Output populates ExitError.Stderr, which is
	// where the skill explains what is actually missing.
	if _, checkErr := exec.Command(binPath, "setup", "--check").Output(); checkErr == nil {
		fmt.Fprintln(out, "  Already configured.")
	} else {
		reportUnconfigured(s, binPath, checkErr, out)
	}

	fmt.Fprintf(out, "\nDone. %s %s installed to:\n  %s\n", s.Name, version, binPath)
	fmt.Fprintf(out, "Skill directory: %s\n", dir)
	return nil
}

// reportUnconfigured explains why the skill is not ready. The skill's own
// message is preferred when it has one — it distinguishes "no credentials"
// from cases the installer cannot diagnose, such as a valid grant on a site
// that does not have the product. The generic hint is only a fallback, and is
// never printed twice.
func reportUnconfigured(s Artifact, binPath string, checkErr error, out io.Writer) {
	var reason string
	if ee, ok := checkErr.(*exec.ExitError); ok {
		reason = strings.TrimSpace(string(ee.Stderr))
	}
	if reason == "" {
		fmt.Fprintln(out, "  Not yet configured.")
		fmt.Fprintf(out, "  Run `%s setup`, or ask Claude to do it for you.\n", s.Name)
		return
	}
	for _, line := range strings.Split(reason, "\n") {
		fmt.Fprintf(out, "  %s\n", line)
	}
	if !strings.Contains(reason, s.Name+" setup") {
		fmt.Fprintf(out, "  Run `%s setup`, or ask Claude to do it for you.\n", s.Name)
	}
}

// runPostInstall invokes the optional per-skill hook. Skills without one exit
// non-zero on the --check probe and are skipped silently; a failing hook is
// non-fatal because the binary is already in place.
func runPostInstall(binPath string, out io.Writer) {
	if err := exec.Command(binPath, "postinstall", "--check").Run(); err != nil {
		return
	}
	fmt.Fprintln(out, "\nRunning post-install checks...")
	cmd := exec.Command(binPath, "postinstall")
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(out, "\n  (Post-install reported issues — see hints above. The binary is")
		fmt.Fprintln(out, "   installed; address the hints and re-run later.)")
	}
}

// linkOnPath makes the skill runnable by bare name. Unix symlinks into
// ~/.local/bin and wires the shell profile if needed; Windows registers the
// bin directory on the user PATH, since symlinks there need elevation.
func linkOnPath(binPath, skill, repo string, out io.Writer) error {
	if runtime.GOOS == "windows" {
		return registerWindowsPath(filepath.Dir(binPath), out)
	}
	userBin, err := userBinDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(userBin, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", userBin, err)
	}
	link := filepath.Join(userBin, skill)
	if err := os.RemoveAll(link); err != nil {
		return fmt.Errorf("replace %s: %w", link, err)
	}
	if err := os.Symlink(binPath, link); err != nil {
		return fmt.Errorf("could not symlink %s: %w", link, err)
	}
	fmt.Fprintf(out, "  Symlinked: %s -> %s\n", link, binPath)

	return ensureOnPath(userBin, skill, repo, out)
}
