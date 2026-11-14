package lint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

const filePrefixAnnotation = "platform.gantry.internal/file-prefix"

// API is one platform offering, read from a Crossplane CompositeResourceDefinition.
type API struct {
	Group      string
	Version    string
	Kind       string
	Namespaced bool
	FilePrefix string
	// Raw is the openAPIV3Schema of the served version, used for unknown-field checks.
	Raw    map[string]any
	schema *jsonschema.Schema
}

func (a API) APIVersion() string { return a.Group + "/" + a.Version }

type xrd struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Annotations map[string]string `yaml:"annotations"`
	} `yaml:"metadata"`
	Spec struct {
		Group string `yaml:"group"`
		Scope string `yaml:"scope"`
		Names struct {
			Kind string `yaml:"kind"`
		} `yaml:"names"`
		Versions []struct {
			Name   string `yaml:"name"`
			Served bool   `yaml:"served"`
			Schema struct {
				OpenAPIV3Schema map[string]any `yaml:"openAPIV3Schema"`
			} `yaml:"schema"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

// LoadAPIs reads every XRD under dir and compiles its schema.
func LoadAPIs(dir string) (map[string]API, error) {
	apis := map[string]API{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var x xrd
		if err := yaml.Unmarshal(data, &x); err != nil || x.Kind != "CompositeResourceDefinition" {
			return nil // not an XRD; compositions live next to them
		}
		prefix := x.Metadata.Annotations[filePrefixAnnotation]
		if prefix == "" {
			return fmt.Errorf("%s: missing %s annotation", path, filePrefixAnnotation)
		}
		for _, v := range x.Spec.Versions {
			if !v.Served {
				continue
			}
			s, err := compile(x.Spec.Names.Kind, v.Schema.OpenAPIV3Schema)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			apis[x.Spec.Names.Kind] = API{
				Group:      x.Spec.Group,
				Version:    v.Name,
				Kind:       x.Spec.Names.Kind,
				Namespaced: x.Spec.Scope != "Cluster",
				FilePrefix: prefix,
				Raw:        v.Schema.OpenAPIV3Schema,
				schema:     s,
			}
		}
		return nil
	})
	return apis, err
}

func compile(kind string, schema map[string]any) (*jsonschema.Schema, error) {
	doc, err := toJSONValue(schema)
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	url := "gantry://" + kind
	if err := c.AddResource(url, doc); err != nil {
		return nil, err
	}
	return c.Compile(url)
}

// toJSONValue normalises YAML-decoded values into what the validator expects
// (json.Number for numbers, map[string]any for objects).
func toJSONValue(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(b))
}
