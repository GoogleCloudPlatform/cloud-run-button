package main

import (
	"reflect"
	"testing"
)

func TestExtractGitLabURL(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    gitHubRepoRef
		wantErr bool
	}{
		{
			name:    "insufficient parts",
			in:      "https://gitlab.com",
			wantErr: true,
		},
		{
			name:    "insufficient parts only username",
			in:      "https://gitlab.com/gitlab-org",
			wantErr: true,
		},
		{
			name: "repository home",
			in:   "https://gitlab.com/gitlab-org/gitlab-runner",
			want: gitHubRepoRef{
				user: "gitlab-org",
				repo: "gitlab-runner",
			},
		},
		{
			name:    "unsupported repo subpath",
			in:      "https://gitlab.com/gitlab-org/gitlab-runner/commits/master",
			wantErr: true,
		},
		{
			name: "repository tree with ref",
			in:   "https://gitlab.com/gitlab-org/gitlab-runner/tree/master",
			want: gitHubRepoRef{
				user: "gitlab-org",
				repo: "gitlab-runner",
				ref:  "master",
			},
		},
		{
			name: "repository tree sub-dir README",
			in:   "https://gitlab.com/gitlab-org/gitlab-runner/tree/v1/sub/dir/README.md",
			want: gitHubRepoRef{
				user: "gitlab-org",
				repo: "gitlab-runner",
				ref:  "v1",
				dir:  "sub/dir",
			},
		},
		{
			name: "repository blob root README",
			in:   "https://gitlab.com/gitlab-org/gitlab-runner/blob/v1/README.md",
			want: gitHubRepoRef{
				user: "gitlab-org",
				repo: "gitlab-runner",
				ref:  "v1",
			},
		},
		{
			name: "repository blob sub-dir README",
			in:   "https://gitlab.com/gitlab-org/gitlab-runner/blob/v1/sub/dir/README.md",
			want: gitHubRepoRef{
				user: "gitlab-org",
				repo: "gitlab-runner",
				ref:  "v1",
				dir:  "sub/dir",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractGitLabURL(mustURL(t, tt.in))
			if (err != nil) != tt.wantErr {
				t.Errorf("extractGitLabURL(%s) error = %v, wantErr %v", tt.in, err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractGitLabURL(%s) got = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}
