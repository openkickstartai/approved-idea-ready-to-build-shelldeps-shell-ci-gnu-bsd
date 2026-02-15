package main

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// Finding represents a single scan result.
type Finding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Command string `json:"command"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

var builtins = map[string]bool{
	"echo": true, "cd": true, "pwd": true, "test": true, "printf": true,
	"read": true, "export": true, "unset": true, "set": true, "shift": true,
	"exit": true, "return": true, "break": true, "continue": true,
	"eval": true, "exec": true, "trap": true, "wait": true, "true": true,
	"false": true, "source": true, "local": true, "declare": true,
	"readonly": true, "getopts": true, "umask": true, "type": true,
	"[": true, "[[": true, "]": true, "]]": true, ":": true,
	"if": true, "then": true, "else": true, "elif": true, "fi": true,
	"for": true, "do": true, "done": true, "while": true, "until": true,
	"case": true, "esac": true, "in": true, "function": true,
}

type compatRule struct {
	cmd string
	re  *regexp.Regexp
	msg string
}

var compatRules = []compatRule{
	{"sed", regexp.MustCompile(`\bsed\b.*\s-i\s+[^'"]`), "sed -i without '' is GNU-only; BSD needs sed -i ''"},
	{"grep", regexp.MustCompile(`\bgrep\b.*\s-P\b`), "grep -P (PCRE) is GNU-only; use -E instead"},
	{"readlink", regexp.MustCompile(`\breadlink\b.*\s-f\b`), "readlink -f is GNU-only; use realpath on macOS"},
	{"date", regexp.MustCompile(`\bdate\b.*\s-d\s`), "date -d is GNU-only; BSD uses date -j -f"},
	{"xargs", regexp.MustCompile(`\bxargs\b.*\s-r\b`), "xargs -r is GNU-only (--no-run-if-empty)"},
	{"stat", regexp.MustCompile(`\bstat\b.*\s-c\b`), "stat -c is GNU-only; BSD uses stat -f"},
}

var (
	reStrip  = regexp.MustCompile(`"[^"]*"|'[^']*'`)
	reSplit  = regexp.MustCompile(`[|;&]+|\$\(`)
	reAssign = regexp.MustCompile(`^[A-Za-z_]\w*=`)
)

// ScanFile scans a shell script file and returns findings.
func ScanFile(path string) ([]Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return scanLines(path, bufio.NewScanner(f))
}

// ScanBytes scans shell script content from a byte slice.
func ScanBytes(name string, data []byte) ([]Finding, error) {
	return scanLines(name, bufio.NewScanner(strings.NewReader(string(data))))
}

func scanLines(path string, sc *bufio.Scanner) ([]Finding, error) {
	var out []Finding
	ln := 0
	for sc.Scan() {
		ln++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		for _, r := range compatRules {
			if r.re.MatchString(raw) {
				out = append(out, Finding{path, ln, r.cmd, "compat", r.msg})
			}
		}
		cleaned := reStrip.ReplaceAllString(raw, "")
		seen := map[string]bool{}
		for _, seg := range reSplit.Split(cleaned, -1) {
			seg = strings.TrimSpace(seg)
			if seg == "" || reAssign.MatchString(seg) {
				continue
			}
			tokens := strings.Fields(seg)
			if len(tokens) == 0 {
				continue
			}
			cmd := tokens[0]
			if i := strings.LastIndex(cmd, "/"); i >= 0 {
				cmd = cmd[i+1:]
			}
			if cmd == "" || cmd[0] == '-' || cmd[0] == '$' || cmd[0] == '(' || cmd[0] == ')' {
				continue
			}
			if !builtins[cmd] && !seen[cmd] {
				seen[cmd] = true
				out = append(out, Finding{path, ln, cmd, "external-dep", "external command: " + cmd})
			}
		}
	}
	return out, sc.Err()
}
