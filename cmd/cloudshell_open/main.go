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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"

	"github.com/briandowns/spinner"
	"github.com/urfave/cli"
)

const (
	flRepoURL        = "repo_url"
	defaultRunRegion = "us-central1"
)

var (
	successPrefix = fmt.Sprintf("[ %s ]", color.New(color.Bold, color.FgGreen).Sprint("✓"))
	errorPrefix   = fmt.Sprintf("[ %s ]", color.New(color.Bold, color.FgRed).Sprint("✖"))
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
	app.Usage = "This tool is only meant to be invoked by Google Cloud Shell"
	app.Description = "Specialized cloudshell_open for the Cloud Run Button"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: flRepoURL,
		},
	}
	app.Action = run
	if err := app.Run(os.Args); err != nil {
		fmt.Printf("%s %+v\n", color.New(color.FgRed, color.Bold).Sprint("Error:"), err)
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

	cmdColor := color.New(color.FgHiBlue)

	repo := c.String(flRepoURL)
	if repo == "" {
		return fmt.Errorf("--%s not specified", flRepoURL)
	}

	highlight := func(s string) string { return color.CyanString(s) }
	parameter := func(s string) string { return color.New(color.FgHiCyan, color.Bold, color.Underline).Sprint(s) }

	end := logProgress("Retrieving your GCP projects...",
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

	end = logProgress(fmt.Sprintf("Cloning git repository %s...", highlight(repo)),
		fmt.Sprintf("Cloned git repository %s.", highlight(repo)),
		fmt.Sprintf("Failed to clone git repository %s", highlight(repo)))
	repoDir, err := handleRepo(repo)
	end(err == nil)
	if err != nil {
		return err
	}

	repoName := filepath.Base(repoDir)
	image := fmt.Sprintf("gcr.io/%s/%s", project, repoName)

	end = logProgress("Building the container image ...",
		"Built the container image.",
		"Failed to build the container image.")
	err = build(repoDir, image)
	end(err == nil)
	if err != nil {
		return err
	}

	end = logProgress(fmt.Sprintf("Pushing the container image %s...", highlight(image)),
		fmt.Sprintf("Pushed container image %s to Google Container Registry.", highlight(image)),
		fmt.Sprintf("Failed to push container image %s to Google Container Registry.", highlight(image)))
	err = push(image)
	end(err == nil)
	if err != nil {
		return fmt.Errorf("failed to push image to %s: %+v", image, err)
	}

	end = logProgress("Deploying the container image to Cloud Run...",
		"Successfully deployed to Cloud Run.",
		"Failed deploying the application to Cloud Run.")
	region := defaultRunRegion
	serviceName := repoName
	url, err := deploy(project, serviceName, image, region)
	end(err == nil)
	if err != nil {
		return err
	}

	fmt.Printf("%s %s %s\n\n",
		successPrefix,
		color.New(color.Bold).Sprint("Your application is now live at URL:"),
		color.New(color.Bold, color.FgGreen, color.Underline).Sprint(url))

	fmt.Println("Make a change to this application:")
	cmdColor.Printf("\tcd %s\n\n", parameter(serviceName))

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
