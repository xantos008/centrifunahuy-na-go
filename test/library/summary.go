//go:build ignore

// Prints the top-level test results of a `go test -json` stream, one line per
// test, sorted: "package<TAB>test<TAB>result". A package that fails without a
// failing test (build error, panic, timeout) prints as "package<TAB>-<TAB>fail".
// Usage: go run test/library/summary.go <file.json>
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: summary <go-test-json>")
		os.Exit(2)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	res := map[string]string{}  // "pkg\ttest" -> result
	failed := map[string]bool{} // packages with a failing top-level test
	pkgFail := map[string]bool{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	for sc.Scan() {
		var e struct{ Action, Package, Test, Output string }
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			fmt.Fprintln(os.Stderr, "invalid go test event:", err)
			os.Exit(1)
		}
		if e.Action == "build-fail" || (e.Action == "output" && strings.TrimSpace(e.Output) == "signal: killed") {
			fmt.Fprintln(os.Stderr, "incomplete library test run:", e.Package, e.Action, e.Output)
			os.Exit(1)
		}
		if e.Action != "pass" && e.Action != "fail" && e.Action != "skip" {
			continue
		}
		pkg := strings.TrimPrefix(strings.TrimPrefix(e.Package, "github.com/centrifugal/centrifuge"), "/")
		if pkg == "" {
			pkg = "."
		}
		switch {
		case e.Test == "":
			if e.Action == "fail" {
				pkgFail[pkg] = true
			}
		case !strings.Contains(e.Test, "/"):
			res[pkg+"\t"+e.Test] = e.Action
			if e.Action == "fail" {
				failed[pkg] = true
			}
		}
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for pkg := range pkgFail {
		if !failed[pkg] {
			res[pkg+"\t-"] = "fail"
		}
	}
	keys := make([]string, 0, len(res))
	for k := range res {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		fmt.Printf("%s\t%s\n", k, res[k])
	}
}
