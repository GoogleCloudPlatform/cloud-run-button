package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"google.golang.org/api/googleapi"
	runapi "google.golang.org/api/run/v1"
)

func optionsToFlags(options options) []string {
	authSetting := "--allow-unauthenticated"
	if options.AllowUnauthenticated != nil && *options.AllowUnauthenticated == false {
		authSetting = "--no-allow-unauthenticated"
	}
	return []string{authSetting}
}

func parseEnv(envs []string) map[string]string {
	out := make(map[string]string)
	for _, v := range envs {
		p := strings.SplitN(v, "=", 2)
		out[p[0]] = p[1]
	}
	return out
}

func deploy(project, name, image, region string, envs []string, options options) (string, error) {
	envVars := parseEnv(envs)

	client, err := runClient(region)
	if err != nil {
		return "", fmt.Errorf("failed to initialize Run API client: %w", err)
	}

	svc, err := getService(project, name, region)
	if err == nil {
		// existing service
		svc = patchService(svc, envVars, image)
		_, err = client.Namespaces.Services.ReplaceService("namespaces/"+project+"/services/"+name, svc).Do()
		if err != nil {
			if e, ok := err.(*googleapi.Error); ok {
				return "", fmt.Errorf("failed to deploy existing Service: code=%d message=%s -- %s", e.Code, e.Message, e.Body)
			}
			return "", fmt.Errorf("failed to deploy to existing Service: %w", err)
		}
	} else {
		// new service
		svc := newService(name, project, image, envVars)
		_, err = client.Namespaces.Services.Create("namespaces/"+project, svc).Do()
		if err != nil {
			if e, ok := err.(*googleapi.Error); ok {
				return "", fmt.Errorf("failed to deploy a new Service: code=%d message=%s -- %s", e.Code, e.Message, e.Body)
			}
			return "", fmt.Errorf("failed to deploy a new Service: %w", err)
		}
	}

	if err := waitReady(project, name, region); err != nil {
		return "", err
	}

	// TODO use 'options' to set --allow-unauthenticated mode

	out, err := getService(project, name, region)
	if err != nil {
		return "", fmt.Errorf("failed to get service after deploying: %w", err)
	}
	return out.Status.Url, nil
}

func newService(name, project, image string, envs map[string]string) *runapi.Service {
	var envVars []*runapi.EnvVar
	for k, v := range envs {
		envVars = append(envVars, &runapi.EnvVar{Name: k, Value: v})
	}

	svc := &runapi.Service{
		ApiVersion: "serving.knative.dev/v1",
		Kind:       "Service",
		Metadata: &runapi.ObjectMeta{
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   project,
		},
		Spec: &runapi.ServiceSpec{
			Template: &runapi.RevisionTemplate{
				Metadata: &runapi.ObjectMeta{
					Name:        generateRevisionName(name, 0),
					Annotations: make(map[string]string),
				},
				Spec: &runapi.RevisionSpec{
					Containers: []*runapi.Container{
						{
							Image: image,
							Env:   envVars,
						},
					},
				},
				ForceSendFields: nil,
				NullFields:      nil,
			},
		},
	}
	applyMeta(svc.Metadata.Annotations, image)
	applyMeta(svc.Spec.Template.Metadata.Annotations, image)

	return svc
}

func applyMeta(meta map[string]string, userImage string) {
	meta["client.knative.dev/user-image"] = userImage
	meta["run.googleapis.com/client-name"] = "cloud-run-button"
}

// generateRevisionName attempts to generate a random revision name that is alphabetically increasing but also has
// a random suffix. objectGeneration is the current object generation.
func generateRevisionName(name string, objectGeneration int64) string {
	num := fmt.Sprintf("%05d", objectGeneration+1)
	out := name + "-" + num + "-"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 3; i++ {
		out += string(int('a') + r.Intn(26))
	}
	return out
}

func patchService(svc *runapi.Service, envs map[string]string, image string) *runapi.Service {
	// merge env vars
	svc.Spec.Template.Spec.Containers[0].Env = mergeEnvs(svc.Spec.Template.Spec.Containers[0].Env, envs)

	// update container image
	svc.Spec.Template.Spec.Containers[0].Image = image

	// apply metadata annotations
	applyMeta(svc.Metadata.Annotations, image)
	applyMeta(svc.Spec.Template.Metadata.Annotations, image)

	// update revision name
	svc.Spec.Template.Metadata.Name = generateRevisionName(svc.Metadata.Name, svc.Metadata.Generation)

	return svc
}

// mergeEnvs updates variables in existing, and adds missing ones.
func mergeEnvs(existing []*runapi.EnvVar, env map[string]string) []*runapi.EnvVar {
	for i, ee := range existing {
		if v, ok := env[ee.Name]; ok {
			existing[i].Value = v
			delete(env, ee.Name)
		}
	}
	// add missing ones
	for k, v := range env {
		existing = append(existing, &runapi.EnvVar{Name: k, Value: v})
	}
	return existing
}

// waitReady waits until the specified service reaches Ready status
func waitReady(project, name, region string) error {
	deadline := time.Now().Add(time.Second * 30)
	for time.Now().Before(deadline) {
		svc, err := getService(project, name, region)
		if err != nil {
			return fmt.Errorf("failed to query Service for readiness: %w", err)
		}

		for _, cond := range svc.Status.Conditions {
			if cond.Type == "Ready" {
				if cond.Status == "True" {
					return nil
				} else if cond.Status == "False" {
					return fmt.Errorf("reason=%s message=%s", cond.Reason, cond.Message)
				}
			}
		}
	}
	return fmt.Errorf("the service did not become ready in 30s, check Cloud Console for logs")
}
