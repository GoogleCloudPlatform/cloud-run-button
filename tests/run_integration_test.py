#!/usr/bin/env python3
import click
import os
import shutil
import subprocess
from urllib import request, error

from googleapiclient.discovery import build as api


GIT_URL = os.environ.get(
    "GIT_URL", "https://github.com/GoogleCloudPlatform/cloud-run-button"
)
GIT_BRANCH = os.environ.get("GIT_BRANCH", "master")
TESTS_DIR = "tests"

# Keep to Python 3.7 systems (gcloud image currently Python 3.7.3)
GOOGLE_CLOUD_PROJECT = os.environ.get("GOOGLE_CLOUD_PROJECT", None)
if not GOOGLE_CLOUD_PROJECT:
    raise Exception("'GOOGLE_CLOUD_PROJECT' env var not found")

GOOGLE_CLOUD_REGION = os.environ.get("GOOGLE_CLOUD_REGION", None)
if not GOOGLE_CLOUD_REGION:
    raise Exception("'GOOGLE_CLOUD_REGION' env var not found")

WORKING_DIR = os.environ.get("WORKING_DIR", ".")

DEBUG=os.environ.get("DEBUG", False)
if DEBUG == "": 
    DEBUG = False

###############################################################################


def debugging(*args):
    c = click.get_current_context()
    output = " ".join([str(k) for k in args])
    if DEBUG:
        print(f"🐞 {output}")


def print_help_msg(command):
    with click.Context(command) as ctx:
        click.echo(command.get_help(ctx))


def gcloud(*args):
    """Invoke the gcloud executable"""
    return run_shell(
        ["gcloud"]
        + list(args)
        + [
            "--platform",
            "managed",
            "--project",
            GOOGLE_CLOUD_PROJECT,
            "--region",
            GOOGLE_CLOUD_REGION,
        ]
    )


def cloudshell_open(directory, repo_url, git_branch):
    """Invoke the cloudshell_open executable."""
    params = [
        f"{WORKING_DIR}/cloudshell_open",
        f"--repo_url={repo_url}",
        f"--git_branch={git_branch}",
    ]

    if directory:
        params += [f"--dir={TESTS_DIR}/{directory}"]
    return run_shell(params)


def run_shell(params):
    """Invoke the given subproceess, capturing output status and returning stdout"""
    debugging("Running:", " ".join(params))

    env = {}
    env.update(os.environ)
    env.update({"TRUSTED_ENVIRONMENT": "true", "SKIP_CLONE_REPORTING": "true"})

    resp = subprocess.run(params, capture_output=True, env=env)

    output = resp.stdout.decode("utf-8")
    error = resp.stderr.decode("utf-8")

    
    if DEBUG:
        # Animated CLIs can make output messy, so only show the long tail
        #debugging("stdout:", output[-300:] or "<None>")
        debugging("stdout:", output or "<None>")
        debugging("stderr:", error or "<None>")

    if resp.returncode != 0:
        raise ValueError(
            f"Command error.\nCommand '{' '.join(params)}' returned {resp.returncode}.\nError: {error}\nOutput: {output}"
        )
    return output


def clean_clone(folder_name):
    """Remove the cloned code"""
    if os.path.isdir(folder_name):
        debugging(f"🟨 Removing old {folder_name} code clone")
        shutil.rmtree(folder_name)


###############################################################################


def deploy_service(directory, repo_url, repo_branch, dirty):
    """Deploy a Cloud Run service using the Cloud Run Button"""

    # The repo name is the last part of a github URL, without the org/user.
    # The service name will be either the directory name, or the repo name
    # The folder name will be the repo name
    repo_name = repo_url.split("/")[-1]
    service_name = directory or repo_name
    folder_name = repo_name

    clean_clone(folder_name)

    if not dirty:
        debugging(f"🟨 Removing old service {service_name} (if it exists)")
        delete_service(service_name)
    else:
        print(f"🙈 Keeping the old service {service_name} (if it exists)")

    print("🟦 Pressing the Cloud Run button...")
    cloudshell_open(directory=directory, repo_url=repo_url, git_branch=repo_branch)

    run = api("run", "v1")
    service_fqdn = f"projects/{GOOGLE_CLOUD_PROJECT}/locations/{GOOGLE_CLOUD_REGION}/services/{service_name}"
    service_obj = run.projects().locations().services().get(name=service_fqdn).execute()

    service_url = service_obj["status"]["url"]

    clean_clone(folder_name)
    return service_url


def delete_service(service_name):
    try:
        gcloud(
            "run",
            "services",
            "delete",
            service_name,
            "--quiet",
        )
        debugging(f"Service {service_name} deleted.")
    except ValueError:
        debugging(f"Service {service_name} not deleted, as it does not exist. ")
        pass


def get_url(service_url, expected_status=200):
    """GET a URL, returning the status and body"""
    debugging(f"Service: {service_url}")

    try:
        resp = request.urlopen(service_url)
        status = resp.status
        body = resp.read().decode("utf-8")

    except error.HTTPError as e:
        # Sometimes errors are OK
        if e.code == expected_status:
            status = e.code
            body = e.msg

    debugging(f"Status: {status}")
    debugging(f"Body: {body[-100:]}")
    return status, body


###############################################################################


@click.group()
def cli() -> None:
    """Tool for testing Cloud Run Button deployments"""
    pass

@cli.command()
@click.option("--description", help="Test description")
@click.option("--repo_url", help="Repo URL to deploy")
@click.option("--repo_branch", default=GIT_BRANCH, help="Branch in Repo URL to deploy")
@click.option("--directory", help="Directory in repo to deploy")
@click.option("--expected_status", default=200, help="Status code to expect")
@click.option("--expected_text", help="Text in service to expect")
@click.option("--dirty", is_flag=True, default=False, help="Keep existing service")
def deploy(
    description,
    directory,
    repo_url,
    repo_branch,
    expected_status,
    expected_text,
    dirty,
):
    """Run service tests.

    Takes a repo url (defaulting to the button's own repo), and an optional directory.
    Deploys the service with the Cloud Run Button, and checks the body and status of the resulting service."""

    if not directory and not repo_url:
        print_help_msg(deploy)
        raise ValueError(
            f"Must supply either a directory for the default repo ({GIT_URL}) or a custom repo.\n"
        )

    if not repo_url:
        repo_url = GIT_URL

    print(
        f"\nRunning {description or directory or 'a test'}\nConfig: {directory or 'root'} in {repo_url} on branch {repo_branch}."
    )
    service_url = deploy_service(directory, repo_url, repo_branch, dirty)
    status, body = get_url(service_url, expected_status)
    print(f"⬜ Service deployed to {service_url}.")

    details = {
        "Service URL": service_url,
        "Expected Status": expected_status,
        "Status": status,
        "Expected": expected_text,
        "Text": body,
    }
    debugging_details = "\n".join([f"{k}: {v}" for k, v in details.items()])

    if expected_status == status:
        print(f"🟢 Service returned expected status {expected_status}.")
    else:

        print(
            f"❌ Service did not return expected status (got {status}, expected {expected_status})."
        )
        raise ValueError(f"Error: Expected status not found:\n{debugging_details}")

    if expected_text:
        if expected_text in body:
            print(f'🟢 Service returned expected content ("{expected_text}").')
        else:
            print(
                f"❌ Service did not return expected content ({expected_text} not in body.\nBody: {body}"
            )
            raise ValueError(f"Error: Expected value not found.\n{debugging_details}")
    print(f"✅ Test successful.")


if __name__ == "__main__":
    cli()
