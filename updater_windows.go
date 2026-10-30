//go:build windows

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

func lockUpdate(target string) (*os.File, error) {
	name := filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".update.lock")
	p, e := windows.UTF16PtrFromString(name)
	if e != nil {
		return nil, e
	}
	// Deny other writers. Duplicating this handle keeps exclusion alive in the helper.
	h, e := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if e != nil {
		return nil, fmt.Errorf("another update owns this installation or the directory is unwritable: %w", e)
	}
	var info windows.ByHandleFileInformation
	if e = windows.GetFileInformationByHandle(h, &info); e != nil || info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		windows.CloseHandle(h)
		return nil, errors.New("unsafe installation lock file")
	}
	return os.NewFile(uintptr(h), name), nil
}
func replaceUpdateFile(from, to string) error {
	src, e := windows.UTF16PtrFromString(from)
	if e != nil {
		return e
	}
	dst, e := windows.UTF16PtrFromString(to)
	if e != nil {
		return e
	}
	return windows.MoveFileEx(src, dst, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
func protectUpdateDirectory(directory string) error {
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return e
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")")
	if e != nil {
		return e
	}
	acl, _, e := sd.DACL()
	if e != nil {
		return e
	}
	return windows.SetNamedSecurityInfo(directory, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
}
func prepareUpdateHandoff(t *updateTransaction) error {
	if e := t.revalidate(); e != nil {
		return e
	}
	executable, e := os.Executable()
	if e != nil {
		return e
	}
	b, e := os.ReadFile(executable)
	if e != nil {
		return e
	}
	helper := filepath.Join(t.Directory, "helper.exe")
	if e = writeUpdateFile(helper, b, 0700); e != nil {
		return e
	}
	var inherited windows.Handle
	if e = windows.DuplicateHandle(windows.CurrentProcess(), windows.Handle(t.lock.Fd()), windows.CurrentProcess(), &inherited, 0, true, windows.DUPLICATE_SAME_ACCESS); e != nil {
		return e
	}
	defer windows.CloseHandle(inherited)
	cmd := exec.Command(helper, "--jutsu-update-helper", t.record(), strconv.FormatUint(uint64(inherited), 10))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{AdditionalInheritedHandles: []syscall.Handle{syscall.Handle(inherited)}}
	if e = cmd.Start(); e != nil {
		return e
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case e := <-done:
			return fmt.Errorf("update helper exited before readiness: %v", e)
		case <-timer.C:
			_ = cmd.Process.Kill()
			<-done
			return errors.New("update helper readiness timed out; retry")
		case <-tick.C:
			if _, e = os.Stat(filepath.Join(t.Directory, "ready")); e == nil {
				return nil
			}
		}
	}
}
func restartUpdated(t *updateTransaction) error {
	// Helper readiness was confirmed before Bubble Tea shutdown. Exit releases the
	// running image; the helper's inherited lock continues excluding other updaters.
	if t.lock != nil {
		t.lock.Close()
		t.lock = nil
	}
	return nil
}
func runUpdateHelper() (bool, error) {
	if len(os.Args) < 2 || os.Args[1] != "--jutsu-update-helper" {
		return false, nil
	}
	if len(os.Args) != 4 {
		return true, errors.New("invalid updater helper invocation")
	}
	handle, e := strconv.ParseUint(os.Args[3], 10, 64)
	if e != nil {
		return true, e
	}
	lock := os.NewFile(uintptr(handle), "update lock")
	defer lock.Close()
	data, e := os.ReadFile(os.Args[2])
	if e != nil {
		return true, e
	}
	var t updateTransaction
	if e = json.Unmarshal(data, &t); e != nil {
		return true, e
	}
	if e = validateUpdateTransaction(&t, os.Args[2]); e != nil {
		return true, e
	}
	executable, e := os.Executable()
	if e != nil {
		return true, e
	}
	if !strings.EqualFold(filepath.Clean(executable), filepath.Join(t.Directory, "helper.exe")) {
		return true, errors.New("helper is outside its update directory")
	}
	parent, e := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(t.Parent))
	if e != nil {
		return true, e
	}
	defer windows.CloseHandle(parent)
	if e = t.revalidate(); e != nil {
		return true, e
	}
	if e = writeUpdateFile(filepath.Join(t.Directory, "ready"), []byte("ready"), 0600); e != nil {
		return true, e
	}
	result, e := windows.WaitForSingleObject(parent, 30000)
	if e != nil || result != windows.WAIT_OBJECT_0 {
		return true, errors.New("update parent did not exit; existing executable unchanged")
	}
	// No current-process image lock remains. The helper never owns the target image.
	err := t.install(updateInstallOps{replace: func(from, to string) error {
		deadline := time.Now().Add(5 * time.Second)
		for {
			err := replaceUpdateFile(from, to)
			if err == nil {
				return nil
			}
			if time.Now().After(deadline) {
				return err
			}
			time.Sleep(100 * time.Millisecond)
		}
	}, launch: func(t *updateTransaction) error {
		cmd := exec.Command(t.Target, t.Args[1:]...)
		cmd.Args = t.Args
		cmd.Dir = t.Cwd
		cmd.Env = t.Env
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return cmd.Start()
	}})
	// A later startup cleans the running helper executable after its image unlocks.
	return true, err
}
