package project

import (
	"regexp"
	"strings"
)

var (
	reqRe     = regexp.MustCompile(`^([A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?)\s*(\[[^\]]*\])?\s*(.*)$`)
	normRe    = regexp.MustCompile(`[-_.]+`)
	simpleVer = regexp.MustCompile(`^(>=|==|~=)\s*([^\s,;]+)$`)
)

// Requirement is a parsed PEP 508 requirement such as "uvicorn[standard]>=0.30".
type Requirement struct {
	Name   string // as written, e.g. "Flask"
	Extras string // e.g. "[standard]"
	Spec   string // everything after the name and extras, e.g. ">=3.0"
}

// Parse splits a requirement string. ok is false for URLs and local paths.
func Parse(req string) (r Requirement, ok bool) {
	req = strings.TrimSpace(req)
	if isDirect(req) {
		return Requirement{}, false
	}
	m := reqRe.FindStringSubmatch(req)
	if m == nil {
		return Requirement{}, false
	}
	return Requirement{Name: m[1], Extras: m[2], Spec: strings.TrimSpace(m[3])}, true
}

func isDirect(req string) bool {
	return strings.Contains(req, "://") ||
		strings.HasPrefix(req, ".") ||
		strings.HasPrefix(req, "/") ||
		strings.HasPrefix(req, "~") ||
		strings.HasSuffix(req, ".whl") ||
		strings.HasSuffix(req, ".tar.gz")
}

// String formats the requirement back to text.
func (r Requirement) String() string { return r.Name + r.Extras + r.Spec }

// Normalize applies PEP 503 name normalization.
func Normalize(name string) string {
	return strings.ToLower(normRe.ReplaceAllString(name, "-"))
}

// Key identifies the package a requirement refers to: the normalized name,
// or the raw string for URLs and paths.
func Key(req string) string {
	if r, ok := Parse(req); ok {
		return Normalize(r.Name)
	}
	return strings.TrimSpace(req)
}

// BumpVersion rewrites a simple ">=X", "==X" or "~=X" spec to version.
// Other specs are returned unchanged.
func BumpVersion(req, version string) string {
	r, ok := Parse(req)
	if !ok {
		return req
	}
	m := simpleVer.FindStringSubmatch(r.Spec)
	if m == nil {
		return req
	}
	r.Spec = m[1] + version
	return r.String()
}
