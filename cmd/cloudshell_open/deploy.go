package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"google.golang.org/api/googleapi"
	runapi "google.golang.org/api/run/v1"
)

// parseEnv parses K=V pairs into a map.
func parseEnv(envs []string) map[string]string {
	out := make(map[string]string)
	for _, v := range envs {
		p := strings.SplitN(v, "=", 2)
		out[p[0]] = p[1]
	}
	return out
}

// deploy reimplements the "gcloud run deploy" command, including setting IAM policy and
// waiting for Service to be Ready.
func deploy(project, name, image, region string, envs []string, options options) (string, error) {
	envVars := parseEnv(envs)

	client, err := runClient(region)
	if err != nil {
		return "", fmt.Errorf("failed to initialize Run API client: %w", err)
	}

	svc, err := getService(project, name, region)
	if err == nil {
		// existing service
		svc = patchService(svc, envVars, image, options)
		_, err = client.Namespaces.Services.ReplaceService("namespaces/"+project+"/services/"+name, svc).Do()
		if err != nil {
			if e, ok := err.(*googleapi.Error); ok {
				return "", fmt.Errorf("failed to deploy existing Service: code=%d message=%s -- %s", e.Code, e.Message, e.Body)
			}
			return "", fmt.Errorf("failed to deploy to existing Service: %w", err)
		}
	} else {
		// new service
		svc := newService(name, project, image, envVars, options)
		_, err = client.Namespaces.Services.Create("namespaces/"+project, svc).Do()
		if err != nil {
			if e, ok := err.(*googleapi.Error); ok {
				return "", fmt.Errorf("failed to deploy a new Service: code=%d message=%s -- %s", e.Code, e.Message, e.Body)
			}
			return "", fmt.Errorf("failed to deploy a new Service: %w", err)
		}
	}

	if options.AllowUnauthenticated == nil || *options.AllowUnauthenticated {
		if err := allowUnauthenticated(project, name, region); err != nil {
			return "", fmt.Errorf("failed to allow unauthenticated requests on the service: %w", err)
		}
	}

	if err := waitReady(project, name, region); err != nil {
		return "", err
	}

	out, err := getService(project, name, region)
	if err != nil {
		return "", fmt.Errorf("failed to get service after deploying: %w", err)
	}
	return out.Status.Url, nil
}

func optionsToResourceRequirements(options options) *runapi.ResourceRequirements {
	limits := make(map[string]string)
	if options.Memory != "" {
		limits["memory"] = options.Memory
	}
	if options.CPU != "" {
		limits["cpu"] = options.CPU
	}
	return &runapi.ResourceRequirements{Limits: limits}
}

func optionsToContainerSpec(options options) *runapi.ContainerPort {
	var containerPortName = "http1"
	if options.HTTP2 != nil && *options.HTTP2 {
		containerPortName = "h2c"
	}

	var containerPort = 8080
	if options.Port > 0 {
		containerPort = options.Port
	}

	return &runapi.ContainerPort{ContainerPort: int64(containerPort), Name: containerPortName}
}

// newService initializes a new Knative Service object with given properties.
func newService(name, project, image string, envs map[string]string, options options) *runapi.Service {
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
							Image:     image,
							Env:       envVars,
							Resources: optionsToResourceRequirements(options),
							Ports:     []*runapi.ContainerPort{optionsToContainerSpec(options)},
						},
					},
				},
				ForceSendFields: nil,
				NullFields:      nil,
			},
		},
	}

	applyMeta(svc.Metadata, image)
	applyMeta(svc.Spec.Template.Metadata, image)

	return svc
}

// applyMeta applies optional annotations to the specified Metadata.Annotation field.
func applyMeta(meta *runapi.ObjectMeta, userImage string) {
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	meta.Annotations["client.knative.dev/user-image"] = userImage
	meta.Annotations["run.googleapis.com/client-name"] = "cloud-run-button"
}

// generateRevisionName attempts to generate a random revision name that is alphabetically increasing but also has
// a random suffix. objectGeneration is the current object generation.
func generateRevisionName(name string, objectGeneration int64) string {
	num := fmt.Sprintf("%05d", objectGeneration+1)
	out := name + "-" + num + "-"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 3; i++ {
		out += string(rune(int('a') + r.Intn(26)))
	}
	return out
}

// patchService modifies an existing Service with requested changes.
func patchService(svc *runapi.Service, envs map[string]string, image string, options options) *runapi.Service {
	// merge env vars
	svc.Spec.Template.Spec.Containers[0].Env = mergeEnvs(svc.Spec.Template.Spec.Containers[0].Env, envs)

	// update container image
	svc.Spec.Template.Spec.Containers[0].Image = image

	// update container port
	svc.Spec.Template.Spec.Containers[0].Ports[0] = optionsToContainerSpec(options)

	// apply metadata annotations
	applyMeta(svc.Metadata, image)
	applyMeta(svc.Spec.Template.Metadata, image)

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
	wait := time.Minute * 4
	deadline := time.Now().Add(wait)
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
		time.Sleep(time.Second * 2)
	}
	return fmt.Errorf("the service did not become ready in %s, check Cloud Console for logs to see why it failed", wait)
}

// allowUnauthenticated sets IAM policy on the specified Cloud Run service to give allUsers subject
// roles/run.invoker role.
func allowUnauthenticated(project, name, region string) error {
	client, err := runapi.NewService(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to initialize Run API client: %w", err)
	}

	res := fmt.Sprintf("projects/%s/locations/%s/services/%s", project, region, name)
	policy, err := client.Projects.Locations.Services.GetIamPolicy(res).Do()
	if err != nil {
		return fmt.Errorf("failed to get IAM policy for Cloud Run Service: %w", err)
	}

	policy.Bindings = append(policy.Bindings, &runapi.Binding{
		Members: []string{"allUsers"},
		Role:    "roles/run.invoker",
	})

	_, err = client.Projects.Locations.Services.SetIamPolicy(res, &runapi.SetIamPolicyRequest{Policy: policy}).Do()
	if err != nil {
		var extra string
		e, ok := err.(*googleapi.Error)
		if ok {
			extra = fmt.Sprintf("code=%d, message=%s -- %s", e.Code, e.Message, e.Body)
		}
		return fmt.Errorf("failed to set IAM policy for Cloud Run Service: %w %s", err, extra)
	}
	return nil
}
