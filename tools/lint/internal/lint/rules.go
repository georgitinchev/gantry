package lint

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Config holds the tunable parts of the policy.
type Config struct {
	// TeamBudget caps the summed size points of everything a team requests.
	TeamBudget int
}

// sizePoints is a deliberately simple cost model: bigger things cost more.
var sizePoints = map[string]int{"small": 1, "medium": 2, "large": 4}

var (
	teamDirRe  = regexp.MustCompile(`^[a-z][a-z0-9-]{1,28}[a-z0-9]$`)
	fileNameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*\.yaml$`)
)

// Check runs every rule and returns the findings.
func Check(apis map[string]API, reqs []*Request, cfg Config) []Finding {
	var fs []Finding
	var valid []*Request
	for _, r := range reqs {
		rf := checkFile(apis, r)
		fs = append(fs, rf...)
		if !HasErrors(rf) {
			valid = append(valid, r)
		}
	}
	// Cross-file rules only look at requests that are individually sound.
	fs = append(fs, checkOwnership(apis, valid)...)
	fs = append(fs, checkUniqueness(valid)...)
	fs = append(fs, checkBudget(valid, cfg)...)
	Sort(fs)
	return fs
}

func checkFile(apis map[string]API, r *Request) []Finding {
	var fs []Finding
	parts := strings.Split(r.Path, "/")
	file := path.Base(r.Path)

	// layout
	if len(parts) != 3 {
		return []Finding{errorf(r.Path, 0, "layout", "requests must live at requests/<team>/<file>.yaml")}
	}
	if !teamDirRe.MatchString(r.Team) {
		fs = append(fs, errorf(r.Path, 0, "layout", "team directory %q must be lowercase kebab-case, 3-30 chars", r.Team))
	}
	if !fileNameRe.MatchString(file) {
		return append(fs, errorf(r.Path, 0, "layout", "file name must be lowercase kebab-case with a .yaml extension"))
	}
	if r.Docs != 1 {
		return append(fs, errorf(r.Path, 0, "single-document", "expected exactly one YAML document, found %d", r.Docs))
	}

	// known API
	api, ok := apis[r.Kind]
	if !ok || r.APIVersion != api.APIVersion() {
		known := make([]string, 0, len(apis))
		for k, a := range apis {
			known = append(known, a.APIVersion()+" "+k)
		}
		sort.Strings(known)
		return append(fs, errorf(r.Path, r.Line("kind"), "known-api",
			"%s %s is not a platform API; available: %s", r.APIVersion, r.Kind, strings.Join(known, ", ")))
	}

	// file naming: <prefix>.yaml or <prefix>-<suffix>.yaml; namespaced kinds use the object name as suffix
	stem := strings.TrimSuffix(file, ".yaml")
	if api.Namespaced {
		if want := api.FilePrefix + "-" + r.Name; stem != want {
			fs = append(fs, errorf(r.Path, r.Line("metadata", "name"), "file-name", "expected file name %s.yaml for %s %q", want, r.Kind, r.Name))
		}
	} else if stem != api.FilePrefix && !strings.HasPrefix(stem, api.FilePrefix+"-") {
		fs = append(fs, errorf(r.Path, 0, "file-name", "%s requests must be named %s.yaml or %s-<suffix>.yaml", r.Kind, api.FilePrefix, api.FilePrefix))
	}

	// metadata and scope
	if r.Name == "" {
		fs = append(fs, errorf(r.Path, r.Line("metadata"), "metadata", "metadata.name is required"))
	}
	if api.Namespaced && r.Namespace == "" {
		fs = append(fs, errorf(r.Path, r.Line("metadata"), "scope", "%s is namespaced: set metadata.namespace to one of your AppEnvironments", r.Kind))
	}
	if !api.Namespaced && r.Namespace != "" {
		fs = append(fs, errorf(r.Path, r.Line("metadata", "namespace"), "scope", "%s is cluster-scoped: remove metadata.namespace", r.Kind))
	}

	// schema
	fs = append(fs, validateSchema(api, r)...)
	if spec, ok := api.Raw["properties"].(map[string]any)["spec"].(map[string]any); ok {
		fs = append(fs, unknownFields(r, spec, r.Spec, []string{"spec"})...)
	}
	return fs
}

func validateSchema(api API, r *Request) []Finding {
	inst, err := toJSONValue(r.Object)
	if err != nil {
		return []Finding{errorf(r.Path, 0, "schema", "%v", err)}
	}
	err = api.schema.Validate(inst)
	var ve *jsonschema.ValidationError
	if err == nil || !errors.As(err, &ve) {
		return nil
	}
	var fs []Finding
	seen := map[string]bool{}
	for _, u := range ve.BasicOutput().Errors {
		if u.Error == nil {
			continue
		}
		msg := u.Error.String()
		if strings.HasPrefix(msg, "validation failed") {
			continue
		}
		loc := strings.Trim(u.InstanceLocation, "/")
		key := loc + msg
		if seen[key] {
			continue
		}
		seen[key] = true
		segs := strings.Split(loc, "/")
		field := strings.ReplaceAll(loc, "/", ".")
		if field == "" {
			field = "(root)"
		}
		fs = append(fs, errorf(r.Path, r.Line(segs...), "schema", "%s: %s", field, msg))
	}
	return fs
}

// unknownFields catches typos the API server would silently prune
// (Kubernetes structural schemas drop unknown fields instead of rejecting them).
func unknownFields(r *Request, schema map[string]any, value any, at []string) []Finding {
	obj, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	if p, _ := schema["x-kubernetes-preserve-unknown-fields"].(bool); p {
		return nil
	}
	props, _ := schema["properties"].(map[string]any)
	var fs []Finding
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		child := append(append([]string{}, at...), k)
		sub, ok := props[k].(map[string]any)
		if !ok {
			allowed := make([]string, 0, len(props))
			for p := range props {
				allowed = append(allowed, p)
			}
			sort.Strings(allowed)
			fs = append(fs, errorf(r.Path, r.Line(child...), "unknown-field",
				"%s is not a field of %s (allowed: %s)", strings.Join(child, "."), r.Kind, strings.Join(allowed, ", ")))
			continue
		}
		fs = append(fs, unknownFields(r, sub, obj[k], child)...)
	}
	return fs
}

func checkOwnership(apis map[string]API, reqs []*Request) []Finding {
	var fs []Finding
	envTeam := map[string]string{} // environment name -> owning team
	for _, r := range reqs {
		if r.Kind != "AppEnvironment" {
			continue
		}
		team, _ := r.Spec["team"].(string)
		envTeam[r.Name] = team
		if team != r.Team {
			fs = append(fs, errorf(r.Path, r.Line("spec", "team"), "ownership", "spec.team %q must match the directory %q", team, r.Team))
		}
		if !strings.HasPrefix(r.Name, r.Team+"-") {
			fs = append(fs, errorf(r.Path, r.Line("metadata", "name"), "ownership", "environment name must start with %q", r.Team+"-"))
		}
	}
	for _, r := range reqs {
		if !apis[r.Kind].Namespaced {
			continue
		}
		owner, ok := envTeam[r.Namespace]
		switch {
		case !ok:
			fs = append(fs, errorf(r.Path, r.Line("metadata", "namespace"), "ownership", "namespace %q is not an AppEnvironment in this repository", r.Namespace))
		case owner != r.Team:
			fs = append(fs, errorf(r.Path, r.Line("metadata", "namespace"), "ownership", "namespace %q belongs to team %q, not %q", r.Namespace, owner, r.Team))
		}
	}
	return fs
}

func checkUniqueness(reqs []*Request) []Finding {
	var fs []Finding
	seen := map[string]string{}
	for _, r := range reqs {
		key := fmt.Sprintf("%s/%s/%s", r.Kind, r.Namespace, r.Name)
		if prev, dup := seen[key]; dup {
			fs = append(fs, errorf(r.Path, r.Line("metadata", "name"), "unique", "%s %q is already defined in %s", r.Kind, r.Name, prev))
			continue
		}
		seen[key] = r.Path
	}
	return fs
}

func checkBudget(reqs []*Request, cfg Config) []Finding {
	if cfg.TeamBudget <= 0 {
		return nil
	}
	used := map[string]int{}
	last := map[string]*Request{}
	for _, r := range reqs {
		size, _ := r.Spec["size"].(string)
		used[r.Team] += sizePoints[size]
		last[r.Team] = r
	}
	var fs []Finding
	for team, pts := range used {
		r := last[team]
		switch {
		case pts > cfg.TeamBudget:
			fs = append(fs, errorf(path.Dir(r.Path), 0, "budget", "team %q uses %d size points, budget is %d (small=1, medium=2, large=4)", team, pts, cfg.TeamBudget))
		case pts == cfg.TeamBudget:
			fs = append(fs, warnf(path.Dir(r.Path), 0, "budget", "team %q is at its budget of %d size points", team, cfg.TeamBudget))
		}
	}
	return fs
}
