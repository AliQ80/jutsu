package main

import "testing"

func TestIsRemoteGitCommand(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{"jj git push", true},
		{"jj git push --remote origin --bookmark main", true},
		{"jj git fetch", true},
		{"jj git clone https://example.com/repo", true},
		{"jj log", false},
		{"jj git init", false},
		{"jj git remote list", false},
		{"jj describe -m hi", false},
	}
	for _, c := range cases {
		if got := isRemoteGitCommand(c.cmd); got != c.want {
			t.Errorf("isRemoteGitCommand(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
}

func TestClassifyFailure(t *testing.T) {
	unknown := "jj: Host key verification failed.\n"
	changed := "@@@@@@@@@@\nWARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!\n@@@@@@@@@@\nHost key verification failed.\n"
	noKnownKey := "No ED25519 host key is known for github.com and you have requested strict checking.\n"
	auth := "fatal: could not read Username for 'https://github.com': terminal prompts disabled\n"
	pubkey := "git@host: Permission denied (publickey).\nfatal: Could not read from remote repository.\n"
	benign := "Error: Non-fast-forwardable update declined\n"

	if !isUnknownHostKey(unknown) {
		t.Error("isUnknownHostKey should match 'Host key verification failed'")
	}
	if !isUnknownHostKey(noKnownKey) {
		t.Error("isUnknownHostKey should match 'no host key is known for'")
	}
	if isUnknownHostKey(changed) {
		t.Error("isUnknownHostKey must NOT match the changed-key (danger) case")
	}
	if !isChangedHostKey(changed) {
		t.Error("isChangedHostKey should match the changed-key warning")
	}
	if !isAuthFailure(auth) {
		t.Error("isAuthFailure should match a credential prompt failure")
	}
	if !isAuthFailure(pubkey) {
		t.Error("isAuthFailure should match a publickey rejection (may be a locked key)")
	}
	if isUnknownHostKey(benign) || isAuthFailure(benign) || isChangedHostKey(benign) {
		t.Error("a benign non-fast-forward error must not match any prompt classifier")
	}
}

func TestSelectedRemoteName(t *testing.T) {
	cases := []struct {
		cmd  string
		want string
	}{
		{"jj git push --remote upstream", "upstream"},
		{"jj git push --remote=upstream", "upstream"},
		{"jj git push --bookmark main", ""},
		{"jj git push", ""},
	}
	for _, c := range cases {
		if got := selectedRemoteName(c.cmd); got != c.want {
			t.Errorf("selectedRemoteName(%q) = %q, want %q", c.cmd, got, c.want)
		}
	}
}

func TestRemoteURL(t *testing.T) {
	list := "origin git@github.com:me/repo.git\nupstream ssh://git@example.com:2222/x.git\n"
	if got := remoteURL(list, "upstream"); got != "ssh://git@example.com:2222/x.git" {
		t.Errorf("named remote = %q", got)
	}
	if got := remoteURL(list, ""); got != "git@github.com:me/repo.git" {
		t.Errorf("default should prefer origin, got %q", got)
	}
	if got := remoteURL("solo git@host:x.git\n", ""); got != "git@host:x.git" {
		t.Errorf("sole remote fallback = %q", got)
	}
}

func TestParseSSHHost(t *testing.T) {
	cases := []struct {
		url  string
		host string
		port string
	}{
		{"git@github.com:me/repo.git", "github.com", ""},
		{"ssh://git@example.com:2222/x.git", "example.com", "2222"},
		{"ssh://git@example.com/x.git", "example.com", ""},
		{"https://github.com/me/repo.git", "", ""}, // https has no host key
		{"git://github.com/me/repo.git", "", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		host, port := parseSSHHost(c.url)
		if host != c.host || port != c.port {
			t.Errorf("parseSSHHost(%q) = (%q, %q), want (%q, %q)", c.url, host, port, c.host, c.port)
		}
	}
}
