package lint

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type Severity string

const (
	SevError   Severity = "error"
	SevWarning Severity = "warning"
)

type Finding struct {
	File     string   `json:"file"`
	Line     int      `json:"line,omitempty"`
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

func errorf(file string, line int, rule, format string, a ...any) Finding {
	return Finding{File: file, Line: line, Rule: rule, Severity: SevError, Message: fmt.Sprintf(format, a...)}
}

func warnf(file string, line int, rule, format string, a ...any) Finding {
	return Finding{File: file, Line: line, Rule: rule, Severity: SevWarning, Message: fmt.Sprintf(format, a...)}
}

func Sort(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		if fs[i].File != fs[j].File {
			return fs[i].File < fs[j].File
		}
		return fs[i].Line < fs[j].Line
	})
}

func HasErrors(fs []Finding) bool {
	for _, f := range fs {
		if f.Severity == SevError {
			return true
		}
	}
	return false
}

// Write renders findings as text, json, or GitHub Actions workflow commands.
func Write(w io.Writer, format string, fs []Finding, checked int) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if fs == nil {
			fs = []Finding{}
		}
		return enc.Encode(map[string]any{"checked": checked, "findings": fs})
	case "github":
		for _, f := range fs {
			fmt.Fprintf(w, "::%s file=%s,line=%d,title=gantry-lint/%s::%s\n",
				f.Severity, f.File, max(f.Line, 1), f.Rule, escape(f.Message))
		}
		fallthrough
	default:
		for _, f := range fs {
			loc := f.File
			if f.Line > 0 {
				loc = fmt.Sprintf("%s:%d", f.File, f.Line)
			}
			if format != "github" {
				fmt.Fprintf(w, "%-7s %s  %s [%s]\n", f.Severity, loc, f.Message, f.Rule)
			}
		}
		errs := 0
		for _, f := range fs {
			if f.Severity == SevError {
				errs++
			}
		}
		fmt.Fprintf(w, "gantry-lint: %d request(s) checked, %d error(s), %d warning(s)\n", checked, errs, len(fs)-errs)
		return nil
	}
}

func escape(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(s)
}
