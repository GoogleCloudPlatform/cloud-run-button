package main

import "net/url"

type repoRef interface {
	GitURL() string
	Dir() string
	Ref() string
}

var (
	availableExtractors = map[string]extractor{
		"github.com": extractGitHubURL,
		"gitlab.com": extractGitLabURL,
	}
)

type extractor func(*url.URL) (repoRef, error)
