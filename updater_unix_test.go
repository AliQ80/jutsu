//go:build !windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateSymlinkAndPermissions(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "actual binary")
	link := filepath.Join(dir, "jutsu")
	if e := os.WriteFile(target, []byte("old"), 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(target, link); e != nil {
		t.Fatal(e)
	}
	tx, e := beginUpdate(link)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.cleanup()
	if tx.Target != target {
		t.Fatalf("target %s", tx.Target)
	}
	if e = tx.stage(context.Background(), []byte("new")); e != nil {
		t.Fatal(e)
	}
	e = tx.install(updateInstallOps{replace: replaceUpdateFile, launch: func(*updateTransaction) error { return nil }})
	if e != nil {
		t.Fatal(e)
	}
	if dest, e := os.Readlink(link); e != nil || dest != target {
		t.Fatal("invocation symlink replaced")
	}
	info, e := os.Stat(target)
	if e != nil || info.Mode().Perm() != 0755 {
		t.Fatalf("permissions changed: %v %v", info, e)
	}
}
func TestUpdateUnwritableInstallation(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses filesystem permission checks")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "jutsu")
	if e := os.WriteFile(target, []byte("old"), 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(dir, 0500); e != nil {
		t.Fatal(e)
	}
	defer os.Chmod(dir, 0700)
	if _, e := beginUpdate(target); e == nil {
		t.Fatal("accepted unwritable directory")
	}
}
func TestUpdateLockRejectsSymlink(t *testing.T) {
	tx := stagedUpdateFixture(t)
	tx.cleanup()
	lockPath := filepath.Join(filepath.Dir(tx.Target), "."+filepath.Base(tx.Target)+".update.lock")
	os.Remove(lockPath)
	victim := filepath.Join(t.TempDir(), "victim")
	os.WriteFile(victim, []byte("safe"), 0600)
	if e := os.Symlink(victim, lockPath); e != nil {
		t.Fatal(e)
	}
	if lock, e := lockUpdate(tx.Target); e == nil {
		lock.Close()
		t.Fatal("followed lock symlink")
	}
}

func TestUpdateUnixExecFailureRestoresBinary(t *testing.T) {
	tx := stagedUpdateFixture(t)
	// The staged fixture is deliberately neither an executable image nor a script.
	// Exercise the actual failing syscall.Exec path, not an injected launch error.
	e := restartUpdated(tx)
	if e == nil {
		t.Fatal("invalid executable unexpectedly launched")
	}
	b, readErr := os.ReadFile(tx.Target)
	if readErr != nil || string(b) != "old" {
		t.Fatalf("failed exec did not restore executable: %q %v", b, readErr)
	}
}
