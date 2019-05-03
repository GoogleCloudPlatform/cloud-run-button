package main

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

var (
	repoPattern = regexp.MustCompile(`^(git@|git://|https://)[a-zA-Z0-9/._:-]*$`)
)

func validRepoURL(repo string) bool { return repoPattern.MatchString(repo) }

func handleRepo(repo string) (string, error) {
	if !validRepoURL(repo) {
		return "", fmt.Errorf("invalid git repo url: %s", repo)
	}
	dir, err := repoDirName(repo)
	if err != nil {
		return "", err
	}
	return dir, clone(repo, dir)
}

func repoDirName(repo string) (string, error) {
	repo = strings.TrimSuffix(repo, ".git")
	i := strings.LastIndex(repo, "/")
	if i == -1 {
		return "", fmt.Errorf("cannot infer directory name from repo %s", repo)
	}
	dir := repo[i+1:]
	if dir == "" {
		return "", fmt.Errorf("cannot parse directory name from repo %s", repo)
	}
	if strings.HasPrefix(dir, ".") {
		return "", fmt.Errorf("attempt to clone into hidden directory: %s", dir)
	}
	return dir, nil
}

func clone(gitRepo, dir string) error {
	cmd := exec.Command("git", "clone", "--", gitRepo, dir)
	log.Printf("Cloning repository %s", gitRepo)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not clone git repository failed: %+v, output:\n%s", err, string(b))
	}
	return nil
}
