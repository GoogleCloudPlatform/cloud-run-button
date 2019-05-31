// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/urfave/cli"
)

const (
	flRepoURL   = "repo_url"
	flGitBranch = "git_branch"
	flSubDir    = "dir"

	defaultRunRegion = "us-central1"
)

var (
	errorLabel    = color.New(color.FgRed, color.Bold)
	successPrefix = fmt.Sprintf("[ %s ]", color.New(color.Bold, color.FgGreen).Sprint("✓"))
	errorPrefix   = fmt.Sprintf("[ %s ]", errorLabel.Sprint("✖"))
	// we have to reset the inherited color first from survey.QuestionIcon
	// see https://github.com/AlecAivazis/survey/issues/193
	questionPrefix = fmt.Sprintf("%s %s ]",
		color.New(color.Reset).Sprint("["),
		color.New(color.Bold, color.FgYellow).Sprint("?"))
	questionSelectFocusIcon = "❯"
)

func main() {
	app := cli.NewApp()
	app.Name = "cloudshell_open"
	app.Usage = "This tool is only meant to be invoked by Google Cloud Shell."
	app.Description = "Specialized cloudshell_open for the Cloud Run Button"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  flRepoURL,
			Usage: "url to git repo",
		},
		cli.StringFlag{
			Name:  flGitBranch,
			Usage: "(optional) branch/revision to use from the git repo",
		},
		cli.StringFlag{
			Name:  flSubDir,
			Usage: "(optional) sub-directory to deploy in the repo",
		},
	}
	app.Action = run
	if err := app.Run(os.Args); err != nil {
		fmt.Printf("%s %+v\n", errorLabel.Sprint("Error:"), err)
		os.Exit(1)
	}
}

func logProgress(msg, endMsg, errMsg string) func(bool) {
	s := spinner.New(spinner.CharSets[9], 300*time.Millisecond)
	s.Prefix = "[ "
	s.Suffix = " ] " + msg
	s.Start()
	return func(success bool) {
		s.Stop()
		if success {
			fmt.Printf("%s %s\n", successPrefix, endMsg)
		} else {
			fmt.Printf("%s %s\n", errorPrefix, errMsg)
		}
	}
}

func run(c *cli.Context) error {
	highlight := func(s string) string { return color.CyanString(s) }
	parameter := func(s string) string { return color.New(color.FgHiCyan, color.Bold, color.Underline).Sprint(s) }
	cmdColor := color.New(color.FgHiBlue)

	repo := c.String(flRepoURL)
	if repo == "" {
		return fmt.Errorf("--%s not specified", flRepoURL)
	}

	if trusted, err := checkCloudShellTrusted(); err != nil {
		return err
	} else if !trusted {
		fmt.Printf("%s You launched this custom Cloud Shell image as \"Do not trust\".\n"+
			"In this mode, your credentials are not available and this experience\n"+
			"cannot deploy to Cloud Run. Start over and \"Trust\" the image.\n", errorLabel.Sprint("Error:"))
		return errors.New("aborting due to untrusted cloud shell environment")
	}

	end := logProgress(fmt.Sprintf("Cloning git repository %s...", highlight(repo)),
		fmt.Sprintf("Cloned git repository %s.", highlight(repo)),
		fmt.Sprintf("Failed to clone git repository %s", highlight(repo)))
	cloneDir, err := handleRepo(repo)
	end(err == nil)
	if err != nil {
		return err
	}

	if gitRev := c.String(flGitBranch); gitRev != "" {
		if err := gitCheckout(cloneDir, gitRev); err != nil {
			return fmt.Errorf("failed to checkout revision %q: %+v", gitRev, err)
		}
	}

	appDir := cloneDir
	if subDir := c.String(flSubDir); subDir != "" {
		// verify if --dir is valid
		appDir = filepath.Join(cloneDir, subDir)
		if fi, err := os.Stat(appDir); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("sub-directory doesn't exist in the cloned repository: %s", appDir)
			}
			return fmt.Errorf("failed to check sub-directory in the repo: %v", err)
		} else if !fi.IsDir() {
			return fmt.Errorf("specified sub-directory path %s is not a directory", appDir)
		}
	}

	appFile, err := getAppFile(appDir)
	if err != nil {
		return fmt.Errorf("error attempting to read the app.json from the cloned repository: %+v", err)
	}
	envs, err := promptEnv(appFile.Env)
	if err != nil {
		return err
	}

	end = logProgress("Retrieving your GCP projects...",
		"Queried list of your GCP projects",
		"Failed to retrieve your GCP projects.",
	)
	projects, err := listProjects()
	end(err == nil)
	if err != nil {
		return err
	}

	project, err := promptProject(projects)
	if err != nil {
		return err
	}

	end = logProgress(
		fmt.Sprintf("Enabling Cloud Run API on project %s...", highlight(project)),
		fmt.Sprintf("Enabled Cloud Run API on project %s.", highlight(project)),
		fmt.Sprintf("Failed to enable required APIs on project %s.", highlight(project)))
	err = enableAPIs(project, []string{"run.googleapis.com", "containerregistry.googleapis.com"})
	end(err == nil)
	if err != nil {
		return err
	}

	repoName := filepath.Base(appDir)
	serviceName := repoName
	if appFile.Name != "" {
		serviceName = appFile.Name
	}
	serviceName = tryFixServiceName(serviceName)

	image := fmt.Sprintf("gcr.io/%s/%s", project, serviceName)
	end = logProgress(fmt.Sprintf("Building container image %s...", highlight(image)),
		fmt.Sprintf("Built container image %s.", highlight(image)),
		"Failed to build container image.")
	err = build(appDir, image)
	end(err == nil)
	if err != nil {
		return err
	}

	end = logProgress("Pushing container image...",
		"Pushed container image to Google Container Registry.",
		"Failed to push container image to Google Container Registry.")
	err = push(image)
	end(err == nil)
	if err != nil {
		return fmt.Errorf("failed to push image to %s: %+v", image, err)
	}

	serviceLabel := highlight(serviceName)
	end = logProgress(fmt.Sprintf("Deploying service %s to Cloud Run...", serviceLabel),
		fmt.Sprintf("Successfully deployed service %s to Cloud Run.", serviceLabel),
		"Failed deploying the application to Cloud Run.")
	region := defaultRunRegion
	url, err := deploy(project, serviceName, image, region, envs)
	end(err == nil)
	if err != nil {
		return err
	}

	fmt.Printf("%s %s %s\n\n",
		successPrefix,
		color.New(color.Bold).Sprint("Your application is now live at URL:"),
		color.New(color.Bold, color.FgGreen, color.Underline).Sprint(url))

	fmt.Println("Make a change to this application:")
	cmdColor.Printf("\tcd %s\n\n", parameter(appDir))

	fmt.Println("Rebuild the container image and push to Container Registry:")
	cmdColor.Printf("\tdocker build -t %s %s\n", parameter(image), parameter("."))
	cmdColor.Printf("\tdocker push %s\n\n\n", parameter(image))

	fmt.Println("Deploy the new version to Cloud Run:")
	cmdColor.Printf("\tgcloud beta run deploy %s\n", parameter(serviceName))
	cmdColor.Printf("\t --project=%s", parameter(project))
	cmdColor.Printf(" \\\n")
	cmdColor.Printf("\t --region=%s", parameter(region))
	cmdColor.Printf(" \\\n")
	cmdColor.Printf("\t --image=%s", parameter(image))
	cmdColor.Printf(" \\\n")
	cmdColor.Printf("\t --allow-unauthenticated\n\n")

	fmt.Printf("Learn more about Cloud Run:\n\t")
	color.New(color.Underline, color.Bold).Println("https://cloud.google.com/run/docs")
	return nil
}

// checkCloudShellTrusted makes an API call to see if the current Cloud Shell
// account is trusted. There's no cleaner way to do this currently
// (bug/134073683). Ideally this would not happen since this image would be trusted,
// but keeping it for dev/test scenarios.
func checkCloudShellTrusted() (bool, error) {
	b, err := exec.Command("gcloud", "organizations", "list", "-q").CombinedOutput()
	if err == nil {
		return true, nil
	}
	if bytes.Contains(b, []byte("PERMISSION_DENIED")) {
		return false, nil
	}
	return false, fmt.Errorf("error determining if cloud shell is trusted: %+v. output=\n%s", err, string(b))
}
