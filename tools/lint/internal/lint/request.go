package lint

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Request is one file under requests/<team>/.
type Request struct {
	Path string // relative to repo root, slash separated
	Team string // directory name
	Docs int    // number of YAML documents in the file

	APIVersion string
	Kind       string
	Name       string
	Namespace  string
	Spec       map[string]any
	Object     map[string]any

	node *yaml.Node
}

// LoadRequests reads every file below root/requests. Parse errors are returned
// as findings so one broken file does not hide problems in the others.
func LoadRequests(root string) ([]*Request, []Finding) {
	var reqs []*Request
	var findings []Finding
	base := filepath.Join(root, "requests")
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if strings.EqualFold(d.Name(), "README.md") {
			return nil
		}
		parts := strings.Split(rel, "/")
		r := &Request{Path: rel}
		if len(parts) == 3 {
			r.Team = parts[1]
		}
		if !strings.HasSuffix(rel, ".yaml") {
			reqs = append(reqs, r) // the layout rule reports it
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, errorf(rel, 0, "parse", "%v", err))
			return nil
		}
		if err := r.parse(data); err != nil {
			findings = append(findings, errorf(rel, 0, "parse", "invalid YAML: %v", err))
			return nil
		}
		reqs = append(reqs, r)
		return nil
	})
	return reqs, findings
}

func (r *Request) parse(data []byte) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var n yaml.Node
		err := dec.Decode(&n)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		r.Docs++
		if r.Docs > 1 {
			continue
		}
		r.node = &n
		var obj map[string]any
		if err := n.Decode(&obj); err != nil {
			return err
		}
		r.Object = obj
		r.APIVersion, _ = obj["apiVersion"].(string)
		r.Kind, _ = obj["kind"].(string)
		if md, ok := obj["metadata"].(map[string]any); ok {
			r.Name, _ = md["name"].(string)
			r.Namespace, _ = md["namespace"].(string)
		}
		r.Spec, _ = obj["spec"].(map[string]any)
	}
	return nil
}

// Line returns the source line of the deepest key found along path,
// falling back to the closest parent, so findings point at the right spot.
func (r *Request) Line(path ...string) int {
	if r.node == nil {
		return 0
	}
	n := r.node
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		n = n.Content[0]
	}
	line := n.Line
	for _, key := range path {
		if n.Kind != yaml.MappingNode {
			break
		}
		found := false
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				line = n.Content[i].Line
				n = n.Content[i+1]
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return line
}
