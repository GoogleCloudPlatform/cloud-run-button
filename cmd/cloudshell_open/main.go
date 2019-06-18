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
	"bufio"
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

	projectCreateURL = "https://console.cloud.google.com/cloud-resource-manager"
)

var (
	linkLabel      = color.New(color.Bold, color.Underline)
	parameterLabel = color.New(color.FgHiYellow, color.Bold, color.Underline)
	errorLabel     = color.New(color.FgRed, color.Bold)
	warningLabel   = color.New(color.Bold, color.FgHiYellow)
	successLabel   = color.New(color.Bold, color.FgGreen)
	successPrefix  = fmt.Sprintf("[ %s ]", successLabel.Sprint("✓"))
	errorPrefix    = fmt.Sprintf("[ %s ]", errorLabel.Sprint("✖"))
	infoPrefix     = fmt.Sprintf("[ %s ]", warningLabel.Sprint("!"))
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
	parameter := func(s string) string { return parameterLabel.Sprint(s) }
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

	var projects []string

	for len(projects) == 0 {
		end = logProgress("Retrieving your GCP projects...",
			"Queried list of your GCP projects",
			"Failed to retrieve your GCP projects.",
		)
		projects, err = listProjects()
		end(err == nil)
		if err != nil {
			return err
		}

		if len(projects) == 0 {
			fmt.Print(errorPrefix+" "+
				warningLabel.Sprint("You don't have any GCP projects to deploy into!")+
				"\n  1. Visit "+linkLabel.Sprint(projectCreateURL),
				"\n  2. Create a new GCP project with a billing account",
				"\n  3. Once you're done, press "+parameterLabel.Sprint("Enter")+" to continue: ")
			bufio.NewReader(os.Stdin).ReadBytes('\n')
		}
	}

	if len(projects) > 1 {
		fmt.Printf(successPrefix+" Found %s projects in your GCP account.\n",
			successLabel.Sprintf("%d", len(projects)))
	}

	project, err := promptProject(projects)
	if err != nil {
		return err
	}

	if err := waitForBilling(project, func(p string) {
		fmt.Print(errorPrefix+" "+
			warningLabel.Sprint("GCP project you chose does not have an active billing account!")+
			"\n  1. Visit "+linkLabel.Sprint(projectCreateURL),
			"\n  2. Associate a billing account for project "+parameterLabel.Sprint(p),
			"\n  3. Once you're done, press "+parameterLabel.Sprint("Enter")+" to continue: ")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}); err != nil {
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

	end = logProgress(fmt.Sprintf("Building container image %s", highlight(image)),
		fmt.Sprintf("Built container image %s", highlight(image)),
		"Failed to build container image.")

	exists, err := dockerFileExists(appDir)
	if err != nil {
		return err
	}

	if exists {
		fmt.Println(infoPrefix + " Attempting to build this application with its Dockerfile...")
		fmt.Println(infoPrefix + " FYI, running the following command:")
		cmdColor.Printf("\tdocker build -t %s %s\n", parameter(image), parameter("."))
		err = dockerBuild(appDir, image)
	} else {
		fmt.Println(infoPrefix + " Attempting to build this application with Cloud Native Buildpacks (buildpacks.io)...")
		fmt.Println(infoPrefix + " FYI, running the following command:")
		cmdColor.Printf("\tpack build %s --path %s --builder heroku/buildacks\n", parameter(image), parameter(appDir))
		err = packBuild(appDir, image)
	}
	end(err == nil)
	if err != nil {
		return fmt.Errorf("this application doesn't have a Dockerfile; attempted to build it via heroku/buildpacks and failed: %s", err)
	}

	fmt.Println(infoPrefix + " FYI, running the following command:")
	cmdColor.Printf("\tdocker push %s\n", parameter(image))
	end = logProgress("Pushing container image...",
		"Pushed container image to Google Container Registry.",
		"Failed to push container image to Google Container Registry.")
	err = dockerPush(image)
	end(err == nil)
	if err != nil {
		return fmt.Errorf("failed to push image to %s: %+v", image, err)
	}

	serviceLabel := highlight(serviceName)
	region := defaultRunRegion

	fmt.Println(infoPrefix + " FYI, running the following command:")
	cmdColor.Printf("\tgcloud beta run deploy %s", parameter(serviceName))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --project=%s", parameter(project))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --region=%s", parameter(region))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --image=%s", parameter(image))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --memory=%s", parameter(defaultRunMemory))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --allow-unauthenticated\n")

	end = logProgress(fmt.Sprintf("Deploying service %s to Cloud Run...", serviceLabel),
		fmt.Sprintf("Successfully deployed service %s to Cloud Run.", serviceLabel),
		"Failed deploying the application to Cloud Run.")
	url, err := deploy(project, serviceName, image, region, envs)
	end(err == nil)
	if err != nil {
		return err
	}

	fmt.Printf("* This application is billed only when it's handling requests.\n")
	fmt.Printf("* Manage this application at Cloud Console:\n\t")
	color.New(color.Underline, color.Bold).Printf("https://console.cloud.google.com/run?project=%s\n", project)
	fmt.Printf("* Learn more about Cloud Run:\n\t")
	color.New(color.Underline, color.Bold).Println("https://cloud.google.com/run/docs")
	fmt.Printf(successPrefix+" %s %s\n",
		color.New(color.Bold).Sprint("Your application is now live here:\n\t"),
		color.New(color.Bold, color.FgGreen, color.Underline).Sprint(url))
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

func waitForBilling(projectID string, prompt func(string)) error {
	for {
		ok, err := checkBillingEnabled(projectID)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		prompt(projectID)
	}
}
