package main

import (
	"github.com/pkg/errors"
	"net/url"
	"strings"
)

const (
	paramDir  = "dir"
	paramRev  = "revision"
	paramRepo = "git_repo"
)

func parseReferer(v string, extractors map[string]extractor) (repoRef, error) {
	u, err := url.Parse(v)
	if err != nil {
		return nil, errors.Errorf("could not parse %s as url", v)
	}
	fn, ok := extractors[u.Hostname()]
	if !ok {
		return nil, errors.Errorf("hostname %s not supported", u.Hostname())
	}

	out, err := fn(u)
	return out, errors.Wrap(err, "failed to extract URL components")
}

func prepURL(r repoRef, overrides url.Values) string {
	u := &url.URL{
		Scheme: "https",
		Host:   "console.cloud.google.com",
		Path:   "cloudshell/editor",
	}
	q := make(url.Values)
	q.Set("cloudshell_image", "gcr.io/cloudrun/button")
	q.Set("shellonly", "true")
	q.Set("cloudshell_git_repo", r.GitURL())
	if v := r.Ref(); v != "" {
		q.Set("cloudshell_git_branch", v)
	}
	if v := r.Dir(); v != "" {
		q.Set("cloudshell_working_dir", v)
	}

	// overrides
	if v := overrides.Get(paramRepo); v != "" {
		q.Set("cloudshell_git_repo", v)
	}
	if v := overrides.Get(paramDir); v != "" {
		q.Set("cloudshell_working_dir", v)
	}
	if v := overrides.Get(paramRev); v != "" {
		q.Set("cloudshell_git_branch", v)
	}

	// pass-through query parameters
	for k := range overrides {
		if strings.HasPrefix(k, "cloudshell_") {
			q.Set(k, overrides.Get(k))
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

type customRepoRef struct{ v url.Values }

func (c customRepoRef) GitURL() string { return c.v.Get(paramRepo) }
func (c customRepoRef) Dir() string    { return c.v.Get(paramDir) }
func (c customRepoRef) Ref() string    { return c.v.Get(paramRev) }
