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
	completePrefix = fmt.Sprintf("[ %s ]", color.New(color.Bold, color.FgGreen).Sprint("✓"))
	errorPrefix    = fmt.Sprintf("[ %s ]", color.New(color.Bold, color.FgRed).Sprint("✖"))
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
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Prefix = "[ "
	s.Suffix = " ] " + msg
	s.Start()
	return func(success bool) {
		s.Stop()
		if success {
			fmt.Printf("%s %s\n", completePrefix,
				color.New(color.Bold).Sprint(endMsg))
		} else {
			fmt.Printf("%s %s\n", errorPrefix, errMsg)
		}
	}
}

func run(c *cli.Context) error {
	repo := c.String(flRepoURL)
	if repo == "" {
		return fmt.Errorf("--%s not specified", flRepoURL)
	}

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

	end = logProgress("Enabling Cloud Run API...",
		"Enabled Cloud Run API.",
		"Failed to enable required APIs on your GCP project %q.")
	err = enableAPIs(project, []string{"run.googleapis.com", "containerregistry.googleapis.com"})
	end(err == nil)
	if err != nil {
		return err
	}

	end = logProgress(fmt.Sprintf("Cloning git repository %s...", repo),
		fmt.Sprintf("Cloned git repository %s.", repo),
		fmt.Sprintf("Failed to clone git repository %s", repo))
	repoDir, err := handleRepo(repo)
	end(err == nil)
	if err != nil {
		return err
	}

	repoName := filepath.Base(repoDir)
	image := fmt.Sprintf("gcr.io/%s/%s", project, repoName)

	end = logProgress("Building container image...",
		"Built container image.",
		"Failed to build the container image.")
	err = build(repoDir, image)
	end(err == nil)
	if err != nil {
		return err
	}

	end = logProgress(fmt.Sprintf("Pushing the container image %s...", image),
		fmt.Sprintf("Pushed container image %s to Google Container Registry.", image),
		fmt.Sprintf("Failed to push container image %s to Google Container Registry.", image))
	err = push(image)
	end(err == nil)
	if err != nil {
		return fmt.Errorf("failed to push image to %s: %+v", image, err)
	}

	end = logProgress("Deploying the container image to Cloud Run...",
		"Application deployed to Cloud Run.",
		"Failed deploying the application to Cloud Run.")
	url, err := deploy(project, repoName, image, defaultRunRegion)
	end(err == nil)
	if err != nil {
		return err
	}
	fmt.Printf("Application now deployed! Visit your URL: %s\n", url)
	return nil
}
