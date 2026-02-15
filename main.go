package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	format := flag.String("format", "text", "output format: text, json")
	check := flag.Bool("check", false, "exit 1 if any finding (CI mode)")
	compatOnly := flag.Bool("compat-only", false, "show only GNU/BSD compat warnings")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: shelldeps [flags] <files or dirs...>\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	var all []Finding
	for _, arg := range flag.Args() {
		fi, err := os.Stat(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
		if fi.IsDir() {
			filepath.Walk(arg, func(p string, info os.FileInfo, e error) error {
				if e != nil || info.IsDir() {
					return nil
				}
				if isShellScript(p) {
					f, _ := ScanFile(p)
					all = append(all, f...)
				}
				return nil
			})
		} else {
			f, err := ScanFile(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				continue
			}
			all = append(all, f...)
		}
	}

	if *compatOnly {
		var filtered []Finding
		for _, f := range all {
			if f.Kind == "compat" {
				filtered = append(filtered, f)
			}
		}
		all = filtered
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(all)
	default:
		for _, f := range all {
			icon := "📦"
			if f.Kind == "compat" {
				icon = "⚠️ "
			}
			fmt.Printf("%s %s:%d  [%s] %s\n", icon, f.File, f.Line, f.Command, f.Message)
		}
	}

	if *check && len(all) > 0 {
		fmt.Fprintf(os.Stderr, "\n❌ %d finding(s) — failing check\n", len(all))
		os.Exit(1)
	}
}

func isShellScript(path string) bool {
	for _, ext := range []string{".sh", ".bash", ".zsh", ".ksh"} {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 64)
	n, _ := f.Read(buf)
	line := string(buf[:n])
	return strings.Contains(line, "#!/bin/sh") ||
		strings.Contains(line, "#!/bin/bash") ||
		strings.Contains(line, "#!/usr/bin/env bash")
}
