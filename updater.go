package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const updateMaxArchive = 128 << 20
const updateMaxBinary = 128 << 20
const updateMaxMetadata = 2 << 20

// Dependencies are per attempt, keeping fixture tests independent of the live API.
type updateSource struct {
	client   *http.Client
	endpoint string
	progress func(int64, int64)
}

func defaultUpdateSource() updateSource {
	return updateSource{client: &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Scheme != "https" || len(via) > 10 {
			return errors.New("unsafe or excessive download redirects")
		}
		return nil
	}}, endpoint: "https://api.github.com/repos/" + releaseRepo + "/releases/tags/"}
}
func updateArchiveName(tag, goos, arch string) (string, error) {
	if (goos != "linux" && goos != "darwin" && goos != "windows") || (arch != "amd64" && arch != "arm64") {
		return "", fmt.Errorf("unsupported platform %s/%s; use the release page", goos, arch)
	}
	if _, _, _, ok := parseSemver(tag); !ok {
		return "", errors.New("invalid release version")
	}
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return "jutsu_" + strings.TrimPrefix(tag, "v") + "_" + goos + "_" + arch + ext, nil
}
func (s updateSource) download(ctx context.Context, address string, limit int64, callbacks ...func(int64, int64)) ([]byte, error) {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" {
		return nil, errors.New("release download requires HTTPS")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "jutsu/"+jutsuVersion)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > limit {
		return nil, errors.New("release download exceeds size limit")
	}
	var reader io.Reader = io.LimitReader(resp.Body, limit+1)
	var counter *updateProgressReader
	if len(callbacks) > 0 && callbacks[0] != nil {
		counter = &updateProgressReader{reader: reader, total: resp.ContentLength, report: callbacks[0]}
		counter.report(0, counter.total)
		reader = counter
	}
	b, err := io.ReadAll(reader)
	if counter != nil {
		counter.report(counter.received, counter.total)
	}
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("release download exceeds size limit")
	}
	return b, nil
}
func verifyUpdateChecksum(manifest, archive []byte, name string) error {
	count := 0
	expected := ""
	for _, line := range strings.Split(string(manifest), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if strings.TrimPrefix(fields[1], "*") == name {
			if len(fields) != 2 {
				return errors.New("malformed checksum entry for " + name)
			}
			count++
			expected = fields[0]
		}
	}
	digest, err := hex.DecodeString(expected)
	if count != 1 || err != nil || len(digest) != sha256.Size {
		return errors.New("expected exactly one valid checksum entry for " + name)
	}
	got := sha256.Sum256(archive)
	if !bytes.Equal(digest, got[:]) {
		return errors.New("release checksum mismatch; existing executable unchanged")
	}
	return nil
}
func extractUpdate(archive []byte, goos string) ([]byte, error) {
	expected := "jutsu"
	if goos == "windows" {
		expected += ".exe"
	}
	var result []byte
	count := 0
	accept := func(name string, regular bool, size int64, r io.Reader) error {
		if strings.Contains(name, "\\") || path.IsAbs(name) || path.Clean(name) != name || name == ".." || strings.HasPrefix(name, "../") {
			return errors.New("unsafe release archive path")
		}
		if !regular {
			return errors.New("release archive contains a link or special entry")
		}
		if size < 0 || size > updateMaxBinary {
			return errors.New("release entry exceeds size limit")
		}
		if name != expected {
			return nil
		}
		count++
		if count != 1 {
			return errors.New("duplicate release executable")
		}
		var err error
		result, err = io.ReadAll(io.LimitReader(r, updateMaxBinary+1))
		if len(result) > updateMaxBinary {
			return errors.New("executable exceeds size limit")
		}
		return err
	}
	if goos == "windows" {
		z, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, err
		}
		for _, f := range z.File {
			if f.FileInfo().IsDir() {
				continue
			}
			r, err := f.Open()
			if err != nil {
				return nil, err
			}
			err = accept(f.Name, f.Mode().IsRegular(), int64(f.UncompressedSize64), r)
			r.Close()
			if err != nil {
				return nil, err
			}
		}
	} else {
		gz, err := gzip.NewReader(bytes.NewReader(archive))
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		tr := tar.NewReader(io.LimitReader(gz, updateMaxArchive+1))
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			if h.Typeflag == tar.TypeDir {
				continue
			}
			if err = accept(h.Name, h.Typeflag == tar.TypeReg, h.Size, tr); err != nil {
				return nil, err
			}
		}
	}
	if count != 1 || len(result) == 0 {
		return nil, errors.New("archive is missing the release executable")
	}
	return result, nil
}
func (s updateSource) binary(ctx context.Context, tag, goos, arch string, progress func(string)) ([]byte, error) {
	name, err := updateArchiveName(tag, goos, arch)
	if err != nil {
		return nil, err
	}
	progress("Downloading release metadata…")
	data, err := s.download(ctx, s.endpoint+url.PathEscape(tag), updateMaxMetadata)
	if err != nil {
		return nil, err
	}
	var release struct {
		Tag    string `json:"tag_name"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err = json.Unmarshal(data, &release); err != nil {
		return nil, err
	}
	if release.Tag != tag {
		return nil, errors.New("release metadata does not match selected version")
	}
	addresses := map[string]string{}
	for _, asset := range release.Assets {
		if asset.Name == name || asset.Name == "checksums.txt" {
			if addresses[asset.Name] != "" {
				return nil, errors.New("ambiguous release assets")
			}
			addresses[asset.Name] = asset.URL
		}
	}
	if addresses[name] == "" || addresses["checksums.txt"] == "" {
		return nil, errors.New("release assets unavailable; use the release page")
	}
	progress("Downloading " + tag + "…")
	archive, err := s.download(ctx, addresses[name], updateMaxArchive, s.progress)
	if err != nil {
		return nil, err
	}
	manifest, err := s.download(ctx, addresses["checksums.txt"], updateMaxMetadata)
	if err != nil {
		return nil, err
	}
	progress("Verifying checksum…")
	if err = verifyUpdateChecksum(manifest, archive, name); err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	progress("Preparing executable…")
	return extractUpdate(archive, goos)
}

type updateTransaction struct {
	Target         string
	Directory      string
	Args           []string
	Env            []string
	Cwd            string
	Parent         int
	Digest         string
	OriginalDigest string
	Committed      bool
	lock           *os.File
	identity       os.FileInfo
}

func (t *updateTransaction) staged() string { return filepath.Join(t.Directory, "new") }
func (t *updateTransaction) backup() string { return filepath.Join(t.Directory, "previous") }
func (t *updateTransaction) record() string { return filepath.Join(t.Directory, "transaction.json") }
func digestFile(name string) (string, error) {
	f, e := os.Open(name)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return "", e
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func writeUpdateFile(name string, b []byte, mode os.FileMode) error {
	f, e := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if e != nil {
		return e
	}
	_, e = f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e == nil {
		e = closeErr
	}
	return e
}
func (t *updateTransaction) save() error {
	b, e := json.Marshal(t)
	if e != nil {
		return e
	}
	tmp := t.record() + ".tmp"
	_ = os.Remove(tmp)
	if e = writeUpdateFile(tmp, b, 0600); e != nil {
		return e
	}
	return replaceUpdateFile(tmp, t.record())
}
func beginUpdate(executable string) (*updateTransaction, error) {
	target, e := filepath.EvalSymlinks(executable)
	if e != nil {
		return nil, e
	}
	target, e = filepath.Abs(target)
	if e != nil {
		return nil, e
	}
	info, e := os.Lstat(target)
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("installed executable is not a regular file")
	}
	lock, e := lockUpdate(target)
	if e != nil {
		return nil, fmt.Errorf("cannot lock installation; another updater may be active or the directory is unwritable. Retry when it finishes, rerun the installer, or choose a writable installation: %w", e)
	}
	dir, e := os.MkdirTemp(filepath.Dir(target), ".jutsu-update-")
	if e != nil {
		lock.Close()
		return nil, fmt.Errorf("installation directory is not writable; rerun the installer or choose a writable location: %w", e)
	}
	if e = protectUpdateDirectory(dir); e != nil {
		os.RemoveAll(dir)
		lock.Close()
		return nil, e
	}
	t := &updateTransaction{Target: target, Directory: dir, Args: append([]string(nil), os.Args...), Env: os.Environ(), Parent: os.Getpid(), lock: lock, identity: info}
	t.Cwd, e = os.Getwd()
	if e != nil {
		t.cleanup()
		return nil, e
	}
	t.OriginalDigest, e = digestFile(target)
	if e != nil {
		t.cleanup()
		return nil, e
	}
	return t, nil
}
func (t *updateTransaction) stage(ctx context.Context, b []byte) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	info, e := os.Stat(t.Target)
	if e != nil {
		return e
	}
	if e = writeUpdateFile(t.staged(), b, info.Mode().Perm()|0700); e != nil {
		return e
	}
	sum := sha256.Sum256(b)
	t.Digest = hex.EncodeToString(sum[:])
	old, e := os.ReadFile(t.Target)
	if e != nil {
		return e
	}
	sumOld := sha256.Sum256(old)
	if hex.EncodeToString(sumOld[:]) != t.OriginalDigest {
		return errors.New("installed executable changed during preparation; retry")
	}
	if e = writeUpdateFile(t.backup(), old, info.Mode().Perm()); e != nil {
		return e
	}
	return t.save()
}
func (t *updateTransaction) cleanup() {
	if t.Directory != "" {
		_ = os.RemoveAll(t.Directory)
	}
	if t.lock != nil {
		_ = t.lock.Close()
		t.lock = nil
	}
}
func (t *updateTransaction) revalidate() error {
	info, e := os.Lstat(t.Target)
	if e != nil {
		return e
	}
	if !info.Mode().IsRegular() || (t.identity != nil && !os.SameFile(t.identity, info)) {
		return errors.New("installed executable changed during update; retry")
	}
	digest, e := digestFile(t.Target)
	if e != nil {
		return e
	}
	if digest != t.OriginalDigest {
		return errors.New("installed executable changed during update; retry")
	}
	for _, name := range []string{t.staged(), t.backup()} {
		info, err := os.Lstat(name)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("update staging contains a nonregular executable")
		}
	}
	staged, e := digestFile(t.staged())
	if e != nil {
		return e
	}
	if staged != t.Digest {
		return errors.New("staged executable changed during update")
	}
	return nil
}
func (t *updateTransaction) rollback(cause error) error { return t.restore(replaceUpdateFile, cause) }
func prepareUpdate(ctx context.Context, source updateSource, executable, tag string, progress func(string)) (*updateTransaction, error) {
	t, e := beginUpdate(executable)
	if e != nil {
		return nil, e
	}
	success := false
	defer func() {
		if !success {
			t.cleanup()
		}
	}()
	b, e := source.binary(ctx, tag, runtime.GOOS, runtime.GOARCH, progress)
	if e != nil {
		return nil, e
	}
	if e = t.stage(ctx, b); e != nil {
		return nil, e
	}
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	success = true
	return t, nil
}
