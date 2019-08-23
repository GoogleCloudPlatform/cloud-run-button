package main

import (
	"net/url"
	"testing"
)

type mockRepo struct{ dir, ref string }

func (m mockRepo) GitURL() string { return "GIT" }

func (m mockRepo) Dir() string { return m.dir }

func (m mockRepo) Ref() string { return m.ref }

func Test_prepURL(t *testing.T) {
	type args struct {
		r         repoRef
		overrides url.Values
	}
	tests := []struct {
		name string
		args args
		want string // TODO(ahmetb): tests may break because on every new go version go map iteration seeds change, and query parameters will shuffle
	}{
		{
			name: "bare repo",
			args: args{
				r:         mockRepo{},
				overrides: nil,
			},
			want: "https://console.cloud.google.com/cloudshell/editor?cloudshell_git_repo=GIT&cloudshell_image=gcr.io%2Fcloudrun%2Fbutton",
		},
		{
			name: " repo with dir",
			args: args{
				r:         mockRepo{dir: "foo"},
				overrides: nil,
			},
			want: "https://console.cloud.google.com/cloudshell/editor?cloudshell_working_dir=foo&cloudshell_git_repo=GIT&cloudshell_image=gcr.io%2Fcloudrun%2Fbutton",
		},
		{
			name: "repo with ref",
			args: args{
				r:         mockRepo{ref: "bar"},
				overrides: nil,
			},
			want: "https://console.cloud.google.com/cloudshell/editor?cloudshell_working_dir=bar&cloudshell_git_repo=GIT&cloudshell_image=gcr.io%2Fcloudrun%2Fbutton",
		},
		{
			name: "passthrough flags",
			args: args{
				r:         mockRepo{},
				overrides: url.Values{
					"cloudshell_xxx":[]string{"yyy"},
				},
			},
			want: "https://console.cloud.google.com/cloudshell/editor?cloudshell_git_repo=GIT&cloudshell_image=gcr.io%2Fcloudrun%2Fbutton&cloudshell_xxx=yyy",
		},
		{
			name: "passthrough flags as override",
			args: args{
				r:         mockRepo{},
				overrides: url.Values{
					"cloudshell_git_repo":[]string{"FOO"},
				},
			},
			want: "https://console.cloud.google.com/cloudshell/editor?cloudshell_git_repo=FOO&cloudshell_image=gcr.io%2Fcloudrun%2Fbutton",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := prepURL(tt.args.r, tt.args.overrides); got != tt.want {
				t.Errorf("prepURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
