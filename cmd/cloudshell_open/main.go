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
	"github.com/AlecAivazis/survey/v2"
	"github.com/GoogleCloudPlatform/cloud-run-button/cmd/instrumentless"
	"google.golang.org/api/transport"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

const (
	flRepoURL       = "repo_url"
	flGitBranch     = "git_branch"
	flSubDir        = "dir"
	flPage          = "page"
	flForceNewClone = "force_new_clone"
	flContext       = "context"

	reauthCredentialsWaitTimeout     = time.Minute * 2
	reauthCredentialsPollingInterval = time.Second

	billingCreateURL    = "https://console.cloud.google.com/billing/create"
	trygcpURL           = "https://console.cloud.google.com/trygcp"
	instrumentlessEvent = "crbutton"
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
	flags.StringVar(&opts.context, flContext, "", "(optional) arbitrary context")

	_ = flags.String(flPage, "", "ignored")
	_ = flags.Bool(flForceNewClone, false, "ignored")
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
	context   string
}

func logProgress(msg, endMsg, errMsg string) func(bool) {
	s := spinner.New(spinner.CharSets[9], 300*time.Millisecond)
	s.Prefix = "[ "
	s.Suffix = " ] " + msg
	s.Start()
	return func(success bool) {
		s.Stop()
		if success {
			if endMsg != "" {
				fmt.Printf("%s %s\n", successPrefix, endMsg)
			}
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

	trusted := os.Getenv("TRUSTED_ENVIRONMENT") == "true"
	if !trusted {
		fmt.Printf("%s You launched this custom Cloud Shell image as \"Do not trust\".\n"+
			"In this mode, your credentials are not available and this experience\n"+
			"cannot deploy to Cloud Run. Start over and \"Trust\" the image.\n", errorLabel.Sprint("Error:"))
		return errors.New("aborting due to untrusted cloud shell environment")
	}

	end := logProgress("Waiting for Cloud Shell authorization...",
		"",
		"Failed to get GCP credentials. Please authorize Cloud Shell if you're presented with a prompt.",
	)
	time.Sleep(time.Second * 2)
	waitCtx, cancelWait := context.WithTimeout(ctx, reauthCredentialsWaitTimeout)
	err := waitCredsAvailable(waitCtx, reauthCredentialsPollingInterval)
	cancelWait()
	end(err == nil)
	if err != nil {
		return err
	}

	end = logProgress(fmt.Sprintf("Cloning git repository %s...", highlight(repo)),
		fmt.Sprintf("Cloned git repository %s.", highlight(repo)),
		fmt.Sprintf("Failed to clone git repository %s", highlight(repo)))
	cloneDir, err := handleRepo(repo)
	if trusted && os.Getenv("SKIP_CLONE_REPORTING") == "" {
		// TODO(ahmetb) had to introduce SKIP_CLONE_REPORTING env var here
		// to skip connecting to :8998 while testing locally if this var is set.
		if err := signalRepoCloneStatus(err == nil); err != nil {
			return err
		}
	}
	end(err == nil)
	if err != nil {
		return err
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

	project := os.Getenv("GOOGLE_CLOUD_PROJECT")

	for project == "" {
		var projects []string

		for len(projects) == 0 {
			end = logProgress("Retrieving your projects...",
				"Queried list of your projects",
				"Failed to retrieve your projects.",
			)
			projects, err = listProjects()
			end(err == nil)
			if err != nil {
				return err
			}

			if len(projects) == 0 {
				fmt.Print(errorPrefix + " " + warningLabel.Sprint("You don't have any projects to deploy into."))
			}
		}

		if len(projects) > 1 {
			fmt.Printf(successPrefix+" Found %s projects in your GCP account.\n",
				successLabel.Sprintf("%d", len(projects)))
		}

		project, err = promptProject(projects)
		if err != nil {
			fmt.Println(errorPrefix + " " + warningLabel.Sprint("You need to create a project"))
			err := promptInstrumentless()
			if err != nil {
				return err
			}
		}
	}

	if err := waitForBilling(project, func(p string) error {
		projectLabel := color.New(color.Bold, color.FgHiCyan).Sprint(project)

		fmt.Println(fmt.Sprintf(errorPrefix+" Project %s does not have an active billing account!", projectLabel))

		billingAccounts, err := billingAccounts()
		if err != nil {
			return fmt.Errorf("could not get billing accounts: %v", err)
		}

		useExisting := false

		if len(billingAccounts) > 0 {
			useExisting, err = prompUseExistingBillingAccount(project)
			if err != nil {
				return err
			}
		}

		if !useExisting {
			err := promptInstrumentless()
			if err != nil {
				return err
			}
		}

		fmt.Println(infoPrefix + " Link the billing account to the project:" +
			"\n  " + linkLabel.Sprintf("https://console.cloud.google.com/billing?project=%s", project))

		fmt.Println(questionPrefix + " " + "Once you're done, press " + parameterLabel.Sprint("Enter") + " to continue: ")

		if _, err := bufio.NewReader(os.Stdin).ReadBytes('\n'); err != nil {
			return err
		}

		// TODO(jamesward) automatically set billing account on project

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

	region := os.Getenv("GOOGLE_CLOUD_REGION")

	if region == "" {
		region, err = promptDeploymentRegion(ctx, project)
		if err != nil {
			return err
		}
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
	inheritedEnv := os.Environ()

	hookEnvs := append([]string{projectEnv, regionEnv, serviceEnv, imageEnv, appDirEnv}, envs...)
	for key, value := range existingEnvVars {
		hookEnvs = append(hookEnvs, fmt.Sprintf("%s=%s", key, value))
	}
	hookEnvs = append(hookEnvs, inheritedEnv...)

	pushImage := true

	if appFile.Hooks.PreBuild.Commands != nil {
		err = runScripts(appDir, appFile.Hooks.PreBuild.Commands, hookEnvs)
	}

	skipBuild := appFile.Build.Skip != nil && *appFile.Build.Skip == true

	skipDocker := false
	skipJib := false

	builderImage := "gcr.io/buildpacks/builder:v1"

	if appFile.Build.Buildpacks.Builder != "" {
		skipDocker = true
		skipJib = true
		builderImage = appFile.Build.Buildpacks.Builder
	}

	dockerFileExists, _ := dockerFileExists(appDir)
	jibMaven, _ := jibMavenConfigured(appDir)

	if skipBuild {
		fmt.Println(infoPrefix + " Skipping built-in build methods")
	} else {
		end = logProgress(fmt.Sprintf("Building container image %s", highlight(image)),
			fmt.Sprintf("Built container image %s", highlight(image)),
			"Failed to build container image.")

		if !skipDocker && dockerFileExists {
			fmt.Println(infoPrefix + " Attempting to build this application with its Dockerfile...")
			fmt.Println(infoPrefix + " FYI, running the following command:")
			cmdColor.Printf("\tdocker build -t %s %s\n", parameter(image), parameter(appDir))
			err = dockerBuild(appDir, image)
		} else if !skipJib && jibMaven {
			pushImage = false
			fmt.Println(infoPrefix + " Attempting to build this application with Jib Maven plugin...")
			fmt.Println(infoPrefix + " FYI, running the following command:")
			cmdColor.Printf("\tmvn package jib:build -Dimage=%s\n", parameter(image))
			err = jibMavenBuild(appDir, image)
		} else {
			fmt.Println(infoPrefix + " Attempting to build this application with Cloud Native Buildpacks (buildpacks.io)...")
			fmt.Println(infoPrefix + " FYI, running the following command:")
			cmdColor.Printf("\tpack build %s --path %s --builder %s\n", parameter(image), parameter(appDir), parameter(builderImage))
			err = packBuild(appDir, image, builderImage)
		}

		end(err == nil)
		if err != nil {
			return fmt.Errorf("attempted to build and failed: %s", err)
		}
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

	deployCommand := "\tgcloud run deploy %s"

	// TODO remove when --use-http2 graduates to GA
	if appFile.Options.HTTP2 != nil && *appFile.Options.HTTP2 == true {
		deployCommand = "\tgcloud beta run deploy %s"
	}

	optionsFlags := optionsToFlags(appFile.Options)

	serviceLabel := highlight(serviceName)
	fmt.Println(infoPrefix + " FYI, running the following command:")
	cmdColor.Printf(deployCommand, parameter(serviceName))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --project=%s", parameter(project))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --platform=%s", parameter("managed"))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --region=%s", parameter(region))
	cmdColor.Println("\\")
	cmdColor.Printf("\t  --image=%s", parameter(image))
	if appFile.Options.Port > 0 {
		cmdColor.Println("\\")
		cmdColor.Printf("\t  --port=%s", parameter(fmt.Sprintf("%d", appFile.Options.Port)))
	}
	if len(envs) > 0 {
		cmdColor.Println("\\")
		cmdColor.Printf("\t  --update-env-vars=%s", parameter(strings.Join(envs, ",")))
	}

	for _, optionFlag := range optionsFlags {
		cmdColor.Println("\\")
		cmdColor.Printf("\t  %s", optionFlag)
	}

	cmdColor.Println("")

	end = logProgress(fmt.Sprintf("Deploying service %s to Cloud Run...", serviceLabel),
		fmt.Sprintf("Successfully deployed service %s to Cloud Run.", serviceLabel),
		"Failed deploying the application to Cloud Run.")
	url, err := deploy(project, serviceName, image, region, envs, appFile.Options)
	end(err == nil)
	if err != nil {
		return err
	}

	hookEnvs = append(hookEnvs, fmt.Sprintf("SERVICE_URL=%s", url))

	if existingService == nil {
		err = runScripts(appDir, appFile.Hooks.PostCreate.Commands, hookEnvs)
		if err != nil {
			return err
		}
	}

	fmt.Printf("* This application is billed only when it's handling requests.\n")
	fmt.Printf("* Manage this application at Cloud Console:\n\t")
	color.New(color.Underline, color.Bold).Printf("https://console.cloud.google.com/run/detail/%s/%s?project=%s\n", region, serviceName, project)
	fmt.Printf("* Learn more about Cloud Run:\n\t")
	color.New(color.Underline, color.Bold).Println("https://cloud.google.com/run/docs")
	fmt.Printf(successPrefix+" %s%s\n",
		color.New(color.Bold).Sprint("Your application is now live here:\n\t"),
		color.New(color.Bold, color.FgGreen, color.Underline).Sprint(url))
	return nil
}

func optionsToFlags(options options) []string {
	var flags []string

	authSetting := "--allow-unauthenticated"
	if options.AllowUnauthenticated != nil && *options.AllowUnauthenticated == false {
		authSetting = "--no-allow-unauthenticated"
	}
	flags = append(flags, authSetting)

	if options.Memory != "" {
		memorySetting := fmt.Sprintf("--memory=%s", options.Memory)
		flags = append(flags, memorySetting)
	}

	if options.CPU != "" {
		cpuSetting := fmt.Sprintf("--cpu=%s", options.CPU)
		flags = append(flags, cpuSetting)
	}

	if options.HTTP2 != nil {
		if *options.HTTP2 == false {
			flags = append(flags, "--no-use-http2")
		} else {
			flags = append(flags, "--use-http2")
		}
	}

	return flags
}

// waitCredsAvailable polls until Cloud Shell VM has available credentials.
// Credentials might be missing in the environment for some GSuite users that
// need to authenticate every N hours. See internal bug 154573156 for details.
func waitCredsAvailable(ctx context.Context, pollInterval time.Duration) error {
	if os.Getenv("SKIP_GCE_CHECK") == "" && !metadata.OnGCE() {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err == context.DeadlineExceeded {
				return errors.New("credentials were not available in the VM, try re-authenticating if Cloud Shell presents an authentication prompt and click the button again")
			}
			return err
		default:
			v, err := metadata.Get("instance/service-accounts/")
			if err != nil {
				return fmt.Errorf("failed to query metadata service to see if credentials are present: %w", err)
			}
			if strings.TrimSpace(v) != "" {
				return nil
			}
			time.Sleep(pollInterval)
		}
	}
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

func instrumentlessCoupon() (*instrumentless.Coupon, error) {
	ctx := context.TODO()

	creds, err := transport.Creds(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get user credentials: %v", err)
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("could not get an auth token: %v", err)
	}

	return instrumentless.GetCoupon(instrumentlessEvent, token.AccessToken)
}

func promptInstrumentless() error {
	coupon, err := instrumentlessCoupon()

	if err != nil || coupon == nil {
		fmt.Println(infoPrefix + " Create a new billing account:")
		fmt.Println("  " + linkLabel.Sprint(billingCreateURL))
		fmt.Println(questionPrefix + " " + "Once you're done, press " + parameterLabel.Sprint("Enter") + " to continue: ")

		if _, err := bufio.NewReader(os.Stdin).ReadBytes('\n'); err != nil {
			return err
		}

		return nil
	}

	code := ""
	parts := strings.Split(coupon.URL, "code=")
	if len(parts) == 2 {
		code = parts[1]
	} else {
		return fmt.Errorf("could not get a coupon code")
	}

	fmt.Println(infoPrefix + " Open this page:\n  " + linkLabel.Sprint(trygcpURL))

	fmt.Println(infoPrefix + " Use this coupon code:\n  " + code)

	fmt.Println(questionPrefix + " Once you're done, press " + parameterLabel.Sprint("Enter") + " to continue: ")

	if _, err := bufio.NewReader(os.Stdin).ReadBytes('\n'); err != nil {
		return err
	}

	return nil
}

func prompUseExistingBillingAccount(project string) (bool, error) {
	useExisting := false

	projectLabel := color.New(color.Bold, color.FgHiCyan).Sprint(project)

	if err := survey.AskOne(&survey.Confirm{
		Default: false,
		Message: fmt.Sprintf("Would you like to use an existing billing account with project %s?", projectLabel),
	}, &useExisting, surveyIconOpts); err != nil {
		return false, fmt.Errorf("could not prompt for confirmation %+v", err)
	}
	return useExisting, nil
}
