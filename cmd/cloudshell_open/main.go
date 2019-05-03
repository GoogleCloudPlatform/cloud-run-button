package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli"
)

const (
	flGitRepo        = "git_repo"
	defaultRunRegion = "us-central1"
)

func init() {
	cwd, _ := os.Getwd()
	log.Printf("pwd=%s", cwd)
}

func main() {
	app := cli.NewApp()
	app.Name = "cloudshell_open"
	app.Description = "specialized cloudshell_open for Deploy to Cloud Run Button"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: flGitRepo,
		},
	}
	app.Action = run
	if err := app.Run(os.Args); err != nil {
		log.Fatalf("failed: %+v", err) // TODO fancier print
	}
}

func run(c *cli.Context) error {
	repo := c.String(flGitRepo)
	if repo == "" {
		return fmt.Errorf("--git_repo not specified")
	}

	log.Printf("Querying a list of your GCP projects.")
	projects, err := listProjects()
	if err != nil {
		return err
	}
	project, err := promptProject(projects)
	if err != nil {
		return err
	}
	log.Printf("chose project: %s", project)

	log.Printf("enabling APIs...")
	if err := enableAPIs(project, []string{"run.googleapis.com", "containerregistry.googleapis.com"}); err != nil {
		return err
	}

	repoDir, err := handleRepo(repo)
	if err != nil {
		return fmt.Errorf("failed to clone the git repo %s: %+v", repo, err)
	}

	repoName := filepath.Base(repoDir)
	image := fmt.Sprintf("gcr.io/%s/%s", project, repoName)

	log.Printf("Building container image from Dockerfile")
	if err := build(repoDir, image); err != nil {
		return fmt.Errorf("failed to build image: %+v", err)
	}
	log.Printf("Pushing the container image to %s", image)
	if err := push(image); err != nil {
		return fmt.Errorf("failed to push image to %s: %+v", image, err)
	}
	log.Printf("Pushed image %s", image)
	log.Printf("Deploying the image to Cloud Run as service %q", repoName)
	url, err := deploy(project, repoName, image, defaultRunRegion)
	if err != nil {
		return fmt.Errorf("deployment failed: %+v", err)
	}
	log.Printf("Application now deployed! Visit your URL: %s", url)
	return nil
}
