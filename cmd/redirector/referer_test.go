package main

import (
	"net/url"
	"reflect"
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
		want string
	}{
		{
			name: "bare repo",
			args: args{
				r:         mockRepo{},
				overrides: nil,
			},
			want: "https://console.cloud.google.com/cloudshell/editor?cloudshell_git_repo=GIT&cloudshell_image=gcr.io%2Fcloudrun%2Fbutton&shellonly=true",
		},
		{
			name: "repo with dir",
			args: args{
				r:         mockRepo{dir: "foo"},
				overrides: nil,
			},
			want: "https://console.cloud.google.com/cloudshell/editor?cloudshell_git_repo=GIT&cloudshell_image=gcr.io%2Fcloudrun%2Fbutton&shellonly=true&cloudshell_working_dir=foo",
		},
		{
			name: "repo with ref",
			args: args{
				r:         mockRepo{ref: "bar"},
				overrides: nil,
			},
			want: "https://console.cloud.google.com/cloudshell/editor?cloudshell_git_branch=bar&cloudshell_git_repo=GIT&cloudshell_image=gcr.io%2Fcloudrun%2Fbutton&shellonly=true",
		},
		{
			name: "repo with slash in ref",
			args: args{
				r:         mockRepo{ref: "bar/quux"},
				overrides: nil,
			},
			want: "https://console.cloud.google.com/cloudshell/editor?cloudshell_git_branch=bar%2Fquux&cloudshell_git_repo=GIT&cloudshell_image=gcr.io%2Fcloudrun%2Fbutton&shellonly=true",
		},
		{
			name: "passthrough flags",
			args: args{
				r: mockRepo{},
				overrides: url.Values{
					"cloudshell_xxx": []string{"yyy"},
				},
			},
			want: "https://console.cloud.google.com/cloudshell/editor?cloudshell_git_repo=GIT&cloudshell_image=gcr.io%2Fcloudrun%2Fbutton&cloudshell_xxx=yyy&shellonly=true",
		},
		{
			name: "passthrough flags as override",
			args: args{
				r: mockRepo{},
				overrides: url.Values{
					"cloudshell_git_repo": []string{"FOO"},
				},
			},
			want: "https://console.cloud.google.com/cloudshell/editor?cloudshell_git_repo=FOO&cloudshell_image=gcr.io%2Fcloudrun%2Fbutton&shellonly=true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := prepURL(tt.args.r, tt.args.overrides)
			got, _ := url.Parse(out)
			want, _ := url.Parse(tt.want)
			if !reflect.DeepEqual(got.Query().Encode(),want.Query().Encode()) {
				t.Errorf("query parameter mismatch prepURL()=\n'%s';\nwant=\n'%s'", got.Query(), want.Query())
			}
			// clear query and compare the rest
			got.RawQuery = ""
			want.RawQuery = ""
			if !reflect.DeepEqual(got,want) {
				t.Errorf("mismatch in rest of url (non-query params) prepURL()=\n'%s';\nwant=\n'%s'", got, want)
			}
		})
	}
}
