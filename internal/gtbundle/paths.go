package gtbundle

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// nameSanitizer keeps a bundle name to a filesystem- and archive-safe token:
// letters, digits, dash, underscore, dot. Everything else collapses to a dash.
var nameSanitizer = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// SanitizeName reduces a proposed bundle name to the safe token used as the
// single top-level directory inside the archive. It never returns "" (falls
// back to "bundle") and strips any leading dots/dashes so the name can't hide
// or look like a flag.
func SanitizeName(s string) string {
	s = nameSanitizer.ReplaceAllString(strings.TrimSpace(s), "-")
	s = strings.Trim(s, ".-")
	if s == "" {
		return "bundle"
	}
	return s
}

// safeLeaf validates a single-path-component filename supplied to the Writer.
// It rejects anything that could escape the intended directory: path
// separators, parent refs, absolute or drive-qualified forms, and a leading
// "~". The returned name is safe to join under a bundle subdirectory.
func safeLeaf(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("gtbundle: empty filename")
	}
	if strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("gtbundle: filename %q must be a single path component", name)
	}
	if strings.Contains(name, "..") {
		return "", fmt.Errorf("gtbundle: filename %q must not contain %q", name, "..")
	}
	if strings.HasPrefix(name, "~") {
		return "", fmt.Errorf("gtbundle: filename %q must not start with %q", name, "~")
	}
	// A Windows drive-qualified leaf ("C:foo") would still be a single
	// component but is not portable/safe.
	if len(name) >= 2 && name[1] == ':' {
		return "", fmt.Errorf("gtbundle: filename %q must not be drive-qualified", name)
	}
	return name, nil
}

// bundlePath composes the archive-relative path for a file: the single
// top-level dir, then the role's subdirectory (if any), then the leaf. It
// always uses forward slashes (tar convention) regardless of host OS.
func bundlePath(topDir string, r Role, leaf string) string {
	if d := r.dir(); d != "" {
		return path.Join(topDir, d, leaf)
	}
	return path.Join(topDir, leaf)
}

// safeMemberPath validates a path read *out* of an archive before it is joined
// to an extraction directory — defense in depth against a crafted tar whose
// member escapes via "..", an absolute path, or a leading top-level that isn't
// the declared name. It returns the cleaned, slash-form relative path.
func safeMemberPath(member string) (string, error) {
	if member == "" {
		return "", fmt.Errorf("gtbundle: empty archive member path")
	}
	// Normalize separators; tar uses forward slashes but be defensive.
	m := strings.ReplaceAll(member, `\`, "/")
	if path.IsAbs(m) || (len(m) >= 2 && m[1] == ':') {
		return "", fmt.Errorf("gtbundle: archive member %q must be relative", member)
	}
	clean := path.Clean(m)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("gtbundle: archive member %q escapes the bundle", member)
	}
	return clean, nil
}
