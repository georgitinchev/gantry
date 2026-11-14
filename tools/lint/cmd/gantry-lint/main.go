// gantry-lint validates self-service requests before they reach a cluster.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/georgitinchev/gantry/tools/lint/internal/lint"
)

func main() {
	root := flag.String("root", ".", "repository root")
	format := flag.String("format", "", "output format: text, json, github (default: github on Actions, else text)")
	budget := flag.Int("team-budget", 6, "max size points per team (0 disables)")
	flag.Parse()

	if *format == "" {
		*format = "text"
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			*format = "github"
		}
	}

	apis, err := lint.LoadAPIs(filepath.Join(*root, "platform", "apis"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "gantry-lint:", err)
		os.Exit(2)
	}
	reqs, findings := lint.LoadRequests(*root)
	findings = append(findings, lint.Check(apis, reqs, lint.Config{TeamBudget: *budget})...)
	lint.Sort(findings)

	if err := lint.Write(os.Stdout, *format, findings, len(reqs)); err != nil {
		fmt.Fprintln(os.Stderr, "gantry-lint:", err)
		os.Exit(2)
	}
	if lint.HasErrors(findings) {
		os.Exit(1)
	}
}
