//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

func lockUpdate(target string) (*os.File, error) {
	name := filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".update.lock")
	fd, e := unix.Open(name, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), name)
	if e = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); e != nil {
		f.Close()
		return nil, fmt.Errorf("another update owns this installation: %w", e)
	}
	return f, nil
}
func replaceUpdateFile(from, to string) error         { return os.Rename(from, to) }
func prepareUpdateHandoff(t *updateTransaction) error { return t.revalidate() }
func runUpdateHelper() (bool, error)                  { return false, nil }
func restartUpdated(t *updateTransaction) error {
	return t.install(updateInstallOps{replace: replaceUpdateFile, launch: func(t *updateTransaction) error {
		if e := os.Chdir(t.Cwd); e != nil {
			return e
		}
		// CLOEXEC releases the lock only after the new image takes over.
		return syscall.Exec(t.Target, t.Args, t.Env)
	}})
}
func protectUpdateDirectory(directory string) error { return os.Chmod(directory, 0700) }
