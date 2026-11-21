package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// UnknownKey is one key in a config document that no setting reads. Load
// decodes leniently, so such a key is dropped without a word: a section
// indented one level too far (recordings: under a trunking system), or a
// misspelled name, leaves the feature at its default while the file looks
// right. The daemon reports these at start-up.
type UnknownKey struct {
	// Line is the key's 1-based line in the document.
	Line int
	// Key is the name as written.
	Key string
	// In is the config path of the section the key was written in, e.g.
	// "trunking.systems[]"; "" for the top level, and "" too when the
	// section's type is used in more than one place.
	In string
	// TopLevel is true when Key names a top-level section but was written
	// somewhere else: the indentation mistake.
	TopLevel bool
}

// String is the operator-facing sentence, without a "config:" prefix.
func (k UnknownKey) String() string {
	where := "at the top level"
	if k.In != "" {
		where = "under " + k.In
	}
	s := fmt.Sprintf("line %d: %q %s is not a setting and is ignored", k.Line, k.Key, where)
	if k.TopLevel {
		return s + fmt.Sprintf(": %q is a top-level section, so it is indented too far here; move it to the left margin", k.Key)
	}
	return s + " (misspelled, or under the wrong section?)"
}

// yamlUnknownField matches yaml.v3's strict-decode complaint.
var yamlUnknownField = regexp.MustCompile(`^line (\d+): field (.+) not found in type (\S+)$`)

// UnknownKeys returns the keys of a config document that Load ignores, in
// document order. It returns nil for a document that does not parse: Load
// reports that.
func UnknownKeys(data []byte) []UnknownKey {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	cfg := Default()
	err := dec.Decode(&cfg)
	var te *yaml.TypeError
	if !errors.As(err, &te) {
		return nil
	}
	paths, top := configKeyIndex()
	var out []UnknownKey
	for _, msg := range te.Errors {
		m := yamlUnknownField.FindStringSubmatch(msg)
		if m == nil {
			continue
		}
		line, _ := strconv.Atoi(m[1])
		k := UnknownKey{Line: line, Key: m[2]}
		if ps := paths[m[3]]; len(ps) == 1 {
			k.In = ps[0]
		}
		k.TopLevel = top[k.Key] && m[3] != reflect.TypeOf(Config{}).String()
		out = append(out, k)
	}
	return out
}

// UnknownKeysInFile is UnknownKeys for the file at path; nil when path is
// empty or unreadable.
func UnknownKeysInFile(path string) []UnknownKey {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return UnknownKeys(data)
}

// configKeyIndex walks the Config type once: paths maps each section
// struct's type name (as yaml.v3 prints it) to the config paths it is used
// at, and top is the set of top-level section names.
func configKeyIndex() (paths map[string][]string, top map[string]bool) {
	paths = map[string][]string{}
	top = map[string]bool{}
	root := reflect.TypeOf(Config{})
	for i := 0; i < root.NumField(); i++ {
		if name := yamlKeyName(root.Field(i)); name != "" {
			top[name] = true
		}
	}
	var walk func(t reflect.Type, path string, depth int)
	walk = func(t reflect.Type, path string, depth int) {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Slice, reflect.Array:
			walk(t.Elem(), path+"[]", depth)
			return
		case reflect.Map:
			walk(t.Elem(), path+"{}", depth)
			return
		case reflect.Struct:
		default:
			return
		}
		if depth > 12 {
			return // a self-referential type; the config has none today
		}
		name := t.String()
		for _, p := range paths[name] {
			if p == path {
				return
			}
		}
		paths[name] = append(paths[name], path)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			key := yamlKeyName(f)
			if key == "" {
				continue
			}
			sub := key
			if path != "" {
				sub = path + "." + key
			}
			walk(f.Type, sub, depth+1)
		}
	}
	walk(root, "", 0)
	return paths, top
}

// yamlKeyName is the key a struct field decodes from: its yaml tag name, or
// yaml.v3's default (the lowercased field name) when untagged. "" for a
// field the decoder skips.
func yamlKeyName(f reflect.StructField) string {
	if !f.IsExported() {
		return ""
	}
	tag, ok := f.Tag.Lookup("yaml")
	if !ok {
		return strings.ToLower(f.Name)
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "-" {
		return ""
	}
	if name == "" {
		return strings.ToLower(f.Name)
	}
	return name
}
