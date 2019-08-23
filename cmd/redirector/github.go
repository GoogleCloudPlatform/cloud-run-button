package main

import (
	"github.com/pkg/errors"
	"net/url"
	"regexp"
	"strings"
)

type gitHubRepoRef struct {
	user string
	repo string
	ref  string
	dir  string
}

func (g gitHubRepoRef) GitURL() string { return "https://github.com/" + g.user + "/" + g.repo + ".git" }
func (g gitHubRepoRef) Dir() string    { return g.dir }
func (g gitHubRepoRef) Ref() string    { return g.ref }

var (
	// ghSubpages matches tree/REF[/SUBPATH] or blob/REF/SUBPATH paths on GitHub.
	ghSubpages = regexp.MustCompile(`(?U)^(tree|blob)\/(.*)?(\/.*)?$`)
)

func extractGitHubURL(u *url.URL) (repoRef, error) {
	var rr gitHubRepoRef
	path := cleanupPath(u.Path)
	parts := strings.SplitN(path, "/", 3)

	if len(parts) < 2 {
		return rr, errors.New("url is not sufficient to infer the repository name")
	}
	rr.user, rr.repo = parts[0], parts[1]

	if len(parts)>2 {
		subPath := parts[2]
		group := ghSubpages.FindStringSubmatch(subPath)
		if len(group) == 0 {
			return rr, errors.New("only tree/ and blob/ urls on the repositories are supported")
		}
		if group[2] != "" {
			rr.ref = group[2]
		}
		rr.dir = strings.TrimLeft(group[3], "/")
	}
	return rr, nil
}

// cleanupPath removes the leading or trailing slashes, or the README.md from the path.
func cleanupPath(path string) string {
	path = strings.TrimSuffix(path, "README.md")
	path = strings.Trim(path, "/")
	return path
}
