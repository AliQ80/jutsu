package main

import "fmt"

// Process creation and filesystem replacement are injectable so failures can be
// exercised without replacing the test runner or starting a second live TUI.
type updateInstallOps struct {
	replace func(string, string) error
	launch  func(*updateTransaction) error
}

func (t *updateTransaction) install(ops updateInstallOps) error {
	if e := t.revalidate(); e != nil {
		t.cleanup()
		return e
	}
	t.Committed = true
	if e := t.save(); e != nil {
		t.cleanup()
		return e
	}
	if e := ops.replace(t.staged(), t.Target); e != nil {
		return t.restore(ops.replace, e)
	}
	if e := ops.launch(t); e != nil {
		return t.restore(ops.replace, e)
	}
	return nil
}
func (t *updateTransaction) restore(replace func(string, string) error, cause error) error {
	if sum, e := digestFile(t.backup()); e != nil || sum != t.OriginalDigest {
		return fmt.Errorf("%v; recovery copy failed verification. Inspect %s and reinstall Jutsu manually at %s", cause, t.Directory, t.Target)
	}
	if e := replace(t.backup(), t.Target); e != nil {
		return fmt.Errorf("%v; restoration failed: %v. Recovery executable: %s; relaunch %s", cause, e, t.backup(), t.Target)
	}
	t.Committed = false
	_ = t.save()
	t.cleanup()
	return fmt.Errorf("%v; previous executable restored. Relaunch %s", cause, t.Target)
}
