package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRedirect_unsupportedMethods(t *testing.T) {
	methods := []string{"PUT", "POST", "DELETE", "OPTIONS", "UNKNOWN"}
	for _, m := range methods {
		req := httptest.NewRequest(m, "/", nil)

		rr := httptest.NewRecorder()
		redirect(rr, req)

		if got, expected := rr.Code, http.StatusMethodNotAllowed; got != expected {
			t.Fatalf("for method=%s got=%d expected=%d", m, got, expected)
		}
	}
}

func TestRedirect_missingReferer(t *testing.T) {
	req := httptest.NewRequest("", "/", nil)
	rr := httptest.NewRecorder()
	redirect(rr, req)
	if expected, status := http.StatusBadRequest, rr.Code; expected != status {
		t.Fatalf("status: got=%d expected=%d", status, expected)
	}
}

func TestRedirect_referer(t *testing.T) {
	repo := "https://github.com/google/new-project"
	req := httptest.NewRequest("", "/", nil)
	req.Header.Set("Referer", repo)

	rr := httptest.NewRecorder()
	redirect(rr, req)

	if expected, status := http.StatusTemporaryRedirect, rr.Code; expected != status {
		t.Fatalf("status: got=%d expected=%d", status, expected)
	}

	loc := rr.Header().Get("location")
	s := "cloudshell_git_repo=" + url.QueryEscape(repo+".git")
	if !strings.Contains(loc, s) {
		t.Fatalf("location header doesn't contain %s\nurl='%s'", s, loc)
	}
}

func TestRedirect_referer_passthrough(t *testing.T) {
	repo := "https://github.com/google/new-project"
	req := httptest.NewRequest("", "/?cloudshell_xxx=yyy", nil)
	req.Header.Set("Referer", repo)

	rr := httptest.NewRecorder()
	redirect(rr, req)

	loc := rr.Header().Get("location")
	s := "cloudshell_xxx=yyy"
	if !strings.Contains(loc, s) {
		t.Fatalf("location header doesn't contain %s\nurl='%s'", s, loc)
	}
}

func TestRedirect_unknownReferer(t *testing.T) {
	req := httptest.NewRequest("", "/", nil)
	req.Header.Set("Referer", "http://example.com")

	rr := httptest.NewRecorder()
	redirect(rr, req)

	if expected, status := http.StatusBadRequest, rr.Code; expected != status {
		t.Fatalf("status: got=%d expected=%d", status, expected)
	}
}

func TestRedirect_queryParams(t *testing.T) {
	req := httptest.NewRequest("", "/?git_repo=foo&dir=bar&revision=staging", nil)
	req.Header.Set("Referer", "http://example.com")

	rr := httptest.NewRecorder()
	redirect(rr, req)
	if expected, status := http.StatusTemporaryRedirect, rr.Code; expected != status {
		t.Fatalf("status: got=%d expected=%d", status, expected)
	}

	loc := rr.Header().Get("location")
	fragments := []string{"cloudshell_git_repo=foo", "cloudshell_working_dir=bar", "cloudshell_git_branch=staging"}
	for _, s := range fragments {
		if !strings.Contains(loc, s) {
			t.Fatalf("location header doesn't contain fragment:%s\nurl='%s'", s, loc)
		}
	}
}
