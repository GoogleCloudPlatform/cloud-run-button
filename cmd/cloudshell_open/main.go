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
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

const (
	flRepoURL   = "repo_url"
	flGitBranch = "git_branch"
	flSubDir    = "dir"
	flPage      = "page"

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

	opts  runOpts
	flags = flag.NewFlagSet("cloudshell_open", flag.ContinueOnError)
)

func init() {
	flags.StringVar(&opts.repoURL, flRepoURL, "", "url to git repo")
	flags.StringVar(&opts.gitBranch, flGitBranch, "", "(optional) branch/revision to use from the git repo")
	flags.StringVar(&opts.subDir, flSubDir, "", "(optional) sub-directory to deploy in the repo")
	_ = flags.String(flPage, "", "ignored")
}
func main() {
	usage := flags.Usage
	flags.Usage = func() {} // control when we print usage string
	if err := flags.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			usage()
			return
		} else {
			fmt.Printf("%s flag parsing issue: %+v\n", warningLabel.Sprint("internal warning:"), err)
		}
	}

	if err := run(opts); err != nil {
		fmt.Printf("%s %+v\n", errorLabel.Sprint("Error:"), err)
		os.Exit(1)
	}
}

type runOpts struct {
	repoURL   string
	gitBranch string
	subDir    string
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

func run(opts runOpts) error {
	ctx := context.Background()
	highlight := func(s string) string { return color.CyanString(s) }
	parameter := func(s string) string { return parameterLabel.Sprint(s) }
	cmdColor := color.New(color.FgHiBlue)

	repo := opts.repoURL
	if repo == "" {
		return fmt.Errorf("--%s not specified", flRepoURL)
	}

	trusted := os.Getenv("TRUSTED_ENVIRONMENT") != "true"
	if !trusted {
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

	if ok, err := hasSubDirsInPATH(cloneDir); err != nil {
		return fmt.Errorf("failed to determine if clone dir has subdirectories in PATH: %v", err)
	} else if ok {
		return fmt.Errorf("cloning git repo to %s could potentially add executable files to PATH", cloneDir)
	}

	if opts.gitBranch != "" {
		if err := gitCheckout(cloneDir, opts.gitBranch); err != nil {
			return fmt.Errorf("failed to checkout revision %q: %+v", opts.gitBranch, err)
		}
	}

	appDir := cloneDir
	if opts.subDir != "" {
		// verify if --dir is valid
		appDir = filepath.Join(cloneDir, opts.subDir)
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
			if _, err := bufio.NewReader(os.Stdin).ReadBytes('\n'); err != nil {
				return err
			}
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

	if err := waitForBilling(project, func(p string) error {
		fmt.Print(errorPrefix+" "+
			warningLabel.Sprint("GCP project you chose does not have an active billing account!")+
			"\n  1. Visit "+linkLabel.Sprint(projectCreateURL),
			"\n  2. Associate a billing account for project "+parameterLabel.Sprint(p),
			"\n  3. Once you're done, press "+parameterLabel.Sprint("Enter")+" to continue: ")
		if _, err := bufio.NewReader(os.Stdin).ReadBytes('\n'); err != nil {
			return err
		}
		return nil
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

	region, err := promptDeploymentRegion(ctx, project)
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

	existingEnvVars := make(map[string]struct{})
	// todo(jamesward) actually determine if the service exists instead of assuming it doesn't if we get an error
	existingService, err := getService(project, serviceName, region)
	if err == nil {
		// service exists
		existingEnvVars, err = envVars(project, serviceName, region)
	}

	neededEnvs := needEnvs(appFile.Env, existingEnvVars)

	envs, err := promptOrGenerateEnvs(neededEnvs)
	if err != nil {
		return err
	}

	projectEnv := fmt.Sprintf("GOOGLE_CLOUD_PROJECT=%s", project)
	regionEnv := fmt.Sprintf("GOOGLE_CLOUD_REGION=%s", region)
	serviceEnv := fmt.Sprintf("K_SERVICE=%s", serviceName)
	imageEnv := fmt.Sprintf("IMAGE_URL=%s", image)
	appDirEnv := fmt.Sprintf("APP_DIR=%s", appDir)
	pathEnv := fmt.Sprintf("PATH=%s", os.Getenv("PATH"))

	hookEnvs := append([]string{projectEnv, regionEnv, serviceEnv, imageEnv, appDirEnv, pathEnv}, envs...)

	pushImage := true

	if appFile.Hooks.PreBuild.Commands != nil {
		err = runScripts(appDir, appFile.Hooks.PreBuild.Commands, hookEnvs)
	}

	skipBuild := appFile.Build.Skip != nil && *appFile.Build.Skip == true

	if skipBuild {
		// do nothing
	} else if dockerFileExists, _ := dockerFileExists(appDir); dockerFileExists {
		fmt.Println(infoPrefix + " Attempting to build this application with its Dockerfile...")
		fmt.Println(infoPrefix + " FYI, running the following command:")
		cmdColor.Printf("\tdocker build -t %s %s\n", parameter(image), parameter("."))
		err = dockerBuild(appDir, image)
	} else if jibMaven, _ := jibMavenConfigured(appDir); jibMaven {
		pushImage = false
		fmt.Println(infoPrefix + " Attempting to build this application with Jib Maven plugin...")
		fmt.Println(infoPrefix + " FYI, running the following command:")
		cmdColor.Printf("\tmvn package jib:build -Dimage=%s\n", parameter(image))
		err = jibMavenBuild(appDir, image)
	} else {
		fmt.Println(infoPrefix + " Attempting to build this application with Cloud Native Buildpacks (buildpacks.io)...")
		fmt.Println(infoPrefix + " FYI, running the following command:")
		cmdColor.Printf("\tpack build %s --path %s --builder heroku/buildpacks\n", parameter(image), parameter(appDir))
		err = packBuild(appDir, image)
	}

	if !skipBuild {
		end = logProgress(fmt.Sprintf("Building container image %s", highlight(image)),
			fmt.Sprintf("Built container image %s", highlight(image)),
			"Failed to build container image.")
	}

	end(err == nil)
	if err != nil {
		return fmt.Errorf("attempted to build and failed: %s", err)
	}

	if appFile.Hooks.PostBuild.Commands != nil {
		err = runScripts(appDir, appFile.Hooks.PostBuild.Commands, hookEnvs)
	}

	if pushImage {
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
	}

	if existingService == nil {
		err = runScripts(appDir, appFile.Hooks.PreCreate.Commands, hookEnvs)
		if err != nil {
			return err
		}
	}

	optionsFlags := optionsToFlags(appFile.Options)

	serviceLabel := highlight(serviceName)
	fmt.Println(infoPrefix + " FYI, running the following command:")
	cmdColor.Printf("\tgcloud run deploy %s", parameter(serviceName))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --project=%s", parameter(project))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --platform=%s", parameter("managed"))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --region=%s", parameter(region))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --image=%s", parameter(image))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --memory=%s", parameter(defaultRunMemory))

	if len(envs) > 0 {
		cmdColor.Println("\\")
		cmdColor.Printf("\t  --update-env-vars=%s", parameter(strings.Join(envs, ",")))
	}

	for _, optionFlag := range optionsFlags {
		cmdColor.Println("\\")
		cmdColor.Printf("\t  %s\n", optionFlag)
	}

	end = logProgress(fmt.Sprintf("Deploying service %s to Cloud Run...", serviceLabel),
		fmt.Sprintf("Successfully deployed service %s to Cloud Run.", serviceLabel),
		"Failed deploying the application to Cloud Run.")
	url, err := deploy(project, serviceName, image, region, envs, appFile.Options)
	end(err == nil)
	if err != nil {
		return err
	}

	if existingService == nil {
		err = runScripts(appDir, appFile.Hooks.PostCreate.Commands, hookEnvs)
		if err != nil {
			return err
		}
	}

	fmt.Printf("* This application is billed only when it's handling requests.\n")
	fmt.Printf("* Manage this application at Cloud Console:\n\t")
	color.New(color.Underline, color.Bold).Printf("https://console.cloud.google.com/run?project=%s\n", project)
	fmt.Printf("* Learn more about Cloud Run:\n\t")
	color.New(color.Underline, color.Bold).Println("https://cloud.google.com/run/docs")
	fmt.Printf(successPrefix+" %s%s\n",
		color.New(color.Bold).Sprint("Your application is now live here:\n\t"),
		color.New(color.Bold, color.FgGreen, color.Underline).Sprint(url))
	return nil
}

func optionsToFlags(options options) []string {
	authSetting := "--allow-unauthenticated"
	if options.AllowUnauthenticated != nil && *options.AllowUnauthenticated == false {
		authSetting = "--no-allow-unauthenticated"
	}
	return []string{authSetting}
}

func waitForBilling(projectID string, prompt func(string) error) error {
	for {
		ok, err := checkBillingEnabled(projectID)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if err := prompt(projectID); err != nil {
			return err
		}
	}
}

// hasSubDirsInPATH determines if anything in PATH is a sub-directory of dir.
func hasSubDirsInPATH(dir string) (bool, error) {
	path := os.Getenv("PATH")
	if path == "" {
		return false, errors.New("PATH is empty")
	}

	paths := strings.Split(path, string(os.PathListSeparator))
	for _, p := range paths {
		ok, err := isSubPath(dir, p)
		if err != nil {
			return false, fmt.Errorf("failure assessing if paths are the same: %v", err)
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// isSubPath determines b is under a. Both paths are evaluated by computing their abs paths.
func isSubPath(a, b string) (bool, error) {
	a, err := filepath.Abs(a)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path for %s: %+v", a, err)
	}
	b, err = filepath.Abs(b)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path for %s: %+v", b, err)
	}
	v, err := filepath.Rel(a, b)
	if err != nil {
		return false, fmt.Errorf("failed to calculate relative path: %v", err)
	}
	return !strings.HasPrefix(v, ".."+string(os.PathSeparator)), nil
}
