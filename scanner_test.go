package main

import (
	"strings"
	"testing"
)

func findCmds(findings []Finding, kind string) map[string]bool {
	m := map[string]bool{}
	for _, f := range findings {
		if f.Kind == kind {
			m[f.Command] = true
		}
	}
	return m
}

func TestExternalDepsDetected(t *testing.T) {
	script := []byte("#!/bin/bash\ncurl -s https://example.com\necho hello\njq '.name' data.json | grep foo\n")
	findings, err := ScanBytes("test.sh", script)
	if err != nil {
		t.Fatal(err)
	}
	cmds := findCmds(findings, "external-dep")
	for _, want := range []string{"curl", "jq", "grep"} {
		if !cmds[want] {
			t.Errorf("expected external dep %q to be detected", want)
		}
	}
	if cmds["echo"] {
		t.Error("echo is a builtin and should NOT be reported")
	}
}

func TestCompatWarnings(t *testing.T) {
	script := []byte("#!/bin/sh\nsed -i s/foo/bar/ file.txt\ngrep -P '\\d+' log.txt\nstat -c '%s' file\n")
	findings, err := ScanBytes("test.sh", script)
	if err != nil {
		t.Fatal(err)
	}
	compat := findCmds(findings, "compat")
	if len(compat) < 3 {
		t.Errorf("expected at least 3 compat warnings, got %d", len(compat))
	}
	for _, want := range []string{"sed", "grep", "stat"} {
		if !compat[want] {
			t.Errorf("expected compat warning for %q", want)
		}
	}
	var sedMsg string
	for _, f := range findings {
		if f.Kind == "compat" && f.Command == "sed" {
			sedMsg = f.Message
		}
	}
	if !strings.Contains(sedMsg, "GNU") {
		t.Error("sed warning should mention GNU")
	}
}

func TestBuiltinsIgnored(t *testing.T) {
	script := []byte("#!/bin/bash\necho hello\ncd /tmp\nexport FOO=bar\nset -e\nif true; then\n  exit 0\nfi\n")
	findings, err := ScanBytes("test.sh", script)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.Kind == "external-dep" {
			t.Errorf("builtin %q should not be reported as external dep", f.Command)
		}
	}
}

func TestPipeChainParsing(t *testing.T) {
	script := []byte("#!/bin/bash\ncat file.txt | sort | uniq -c | head -10\n")
	findings, err := ScanBytes("test.sh", script)
	if err != nil {
		t.Fatal(err)
	}
	cmds := findCmds(findings, "external-dep")
	for _, want := range []string{"cat", "sort", "uniq", "head"} {
		if !cmds[want] {
			t.Errorf("expected %q in pipe chain to be detected", want)
		}
	}
}

func TestFullPathCommand(t *testing.T) {
	script := []byte("#!/bin/sh\n/usr/bin/curl http://example.com\n/usr/local/bin/jq .\n")
	findings, err := ScanBytes("test.sh", script)
	if err != nil {
		t.Fatal(err)
	}
	cmds := findCmds(findings, "external-dep")
	if !cmds["curl"] || !cmds["jq"] {
		t.Errorf("full-path commands should be detected: got %v", cmds)
	}
}
