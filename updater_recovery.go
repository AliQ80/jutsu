package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validateUpdateTransaction(t *updateTransaction, record string) error {
	if !filepath.IsAbs(t.Target) || !filepath.IsAbs(t.Directory) || !filepath.IsAbs(t.Cwd) || len(t.Args) == 0 || t.Parent <= 0 {
		return errors.New("invalid update transaction")
	}
	if filepath.Clean(record) != t.record() || filepath.Dir(t.Directory) != filepath.Dir(t.Target) || !strings.HasPrefix(filepath.Base(t.Directory), ".jutsu-update-") {
		return errors.New("update paths are outside the installation")
	}
	for _, name := range []string{t.Directory, t.record(), t.backup()} {
		info, e := os.Lstat(name)
		if e != nil {
			return e
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("update transaction contains a symlink")
		}
	}
	if sum, e := digestFile(t.backup()); e != nil || sum != t.OriginalDigest {
		return errors.New("invalid recovery executable")
	}
	return nil
}

// Only clean transactions for this executable, under an installation lock.
// Unknown or interrupted commits are retained and surfaced for manual recovery.
func inspectUpdateRecovery(executable string) []string {
	target, e := filepath.EvalSymlinks(executable)
	if e != nil {
		return nil
	}
	target, e = filepath.Abs(target)
	if e != nil {
		return nil
	}
	entries, e := os.ReadDir(filepath.Dir(target))
	if e != nil {
		return nil
	}
	var notices []string
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), ".jutsu-update-") {
			continue
		}
		dir := filepath.Join(filepath.Dir(target), entry.Name())
		record := filepath.Join(dir, "transaction.json")
		info, e := os.Lstat(dir)
		if e != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		data, e := os.ReadFile(record)
		if e != nil {
			continue
		}
		var t updateTransaction
		if json.Unmarshal(data, &t) != nil || t.Target != target || t.Directory != dir {
			continue
		}
		lock, e := lockUpdate(target)
		if e != nil {
			continue
		}
		t.lock = lock
		if e = validateUpdateTransaction(&t, record); e != nil {
			lock.Close()
			notices = append(notices, fmt.Sprintf("Update recovery data needs inspection: %s (%v)", dir, e))
			continue
		}
		sum, e := digestFile(target)
		if e == nil && ((!t.Committed && sum == t.OriginalDigest) || (t.Committed && sum == t.Digest)) {
			t.cleanup()
			continue
		}
		lock.Close()
		notices = append(notices, fmt.Sprintf("Interrupted update: recovery executable %s; installed executable %s. Reinstall a release or restore the recovery copy manually.", t.backup(), target))
	}
	return notices
}
