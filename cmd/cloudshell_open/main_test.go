package main

import (
	"fmt"
	"os"
	"testing"
)

func Test_isSubPath(t *testing.T) {
	type args struct {
		a string
		b string
	}
	tests := []struct {
		args    args
		want    bool
		wantErr bool
	}{
		{
			args: args{"/foo", "/foo/bar"},
			want: true,
		},
		{
			args: args{"/foo", "/foo"},
			want: true,
		},
		{
			args: args{"/foo/bar", "/foo/quux"},
			want: false,
		},
		{
			args: args{"/foo/bar", "/foo/bar/../bar/quux"},
			want: true,
		},
		{
			args: args{"/foo/../foo", "/foo/bar"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%#v", tt.args), func(t *testing.T) {
			got, err := isSubPath(tt.args.a, tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("isSubPath(%s,%s) error = %v, wantErr %v", tt.args.a, tt.args.b, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isSubPath(%s,%s) got = %v, want %v", tt.args.a, tt.args.b, got, tt.want)
			}
		})
	}
}

func Test_hasSubDirsInPATH(t *testing.T) {
	type args struct {
		path string
		dir  string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "not in path",
			args: args{
				path: "/a:/b",
				dir:  "/c",
			},
			want: false,
		},
		{
			name: "not in path as it is a parent dir",
			args: args{
				path: "/usr/bin:/usr/local/bin",
				dir:  "/usr/foo",
			},
			want: false,
		},
		{
			name: "has subdir in path",
			args: args{
				path: "/foo/bar/quux:/b",
				dir:  "/foo",
			},
			want: true,
		},
		{
			name: "in path, with non-cleaned paths",
			args: args{
				path: "/foo/../foo:/quux",
				dir:  "/foo/bar",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := os.Getenv("PATH")
			os.Setenv("PATH", tt.args.path)
			got, err := hasSubDirsInPATH(tt.args.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("hasSubDirsInPATH(%s) error = %v, wantErr %v", tt.args.dir, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("hasSubDirsInPATH(%s) got = %v, want %v", tt.args.dir, got, tt.want)
			}
			os.Setenv("PATH", op)
		})
	}
}
