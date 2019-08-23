package main

import (
	"net/url"
	"reflect"
	"testing"
)

func Test_cleanupPath(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		{
			name: "empty",
			path: "",
			want: "",
		},
		{
			name: "root",
			path: "/",
			want: "",
		},
		{
			name: "slashes",
			path: "/user/repo/tree/master/",
			want: "user/repo/tree/master",
		},
		{
			name: "readme.md trimming",
			path: "/path/README.md",
			want: "path",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanupPath(tt.path); got != tt.want {
				t.Errorf("cleanupPath(%s) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestExtractGitHubURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    gitHubRepoRef
		wantErr bool
	}{
		{
			name:    "insufficient parts",
			url:     "https://github.com",
			wantErr: true,
		},
		{
			name:    "insufficient parts only username",
			url:     "https://github.com/google",
			wantErr: true,
		},
		{
			name: "repository home",
			url:  "https://github.com/google/new-project",
			want: gitHubRepoRef{
				user: "google",
				repo: "new-project",
			},
		},
		{
			name:    "unsupported repository subpath",
			url:     "https://github.com/google/new-project/commits/master",
			wantErr: true,
		},
		{
			name: "repository tree with ref",
			url:  "https://github.com/google/new-project/tree/master",
			want: gitHubRepoRef{
				user: "google",
				repo: "new-project",
				ref:  "master",
			},
		},
		{
			name: "repository tree sub-dir README",
			url:  "https://github.com/google/new-project/tree/v1/sub/dir/README.md",
			want: gitHubRepoRef{
				user: "google",
				repo: "new-project",
				ref:  "v1",
				dir:  "sub/dir",
			},
		},
		{
			name: "repository blob root README",
			url:  "https://github.com/google/new-project/blob/v1/README.md",
			want: gitHubRepoRef{
				user: "google",
				repo: "new-project",
				ref:  "v1",
			},
		},
		{
			name: "repository blob sub-dir README",
			url:  "https://github.com/google/new-project/blob/v1/sub/dir/README.md",
			want: gitHubRepoRef{
				user: "google",
				repo: "new-project",
				ref:  "v1",
				dir:  "sub/dir",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractGitHubURL(mustURL(t, tt.url))
			if (err != nil) != tt.wantErr {
				t.Errorf("extractGitHubURL(%s) error = %v, wantErr %v", tt.url, err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractGitHubURL(%s) got = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func mustURL(t *testing.T, u string) *url.URL {
	t.Helper()
	v, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
