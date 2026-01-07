// A generated module for BlogCi functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/blog-ci/internal/dagger"
	"fmt"
	"math"
	"math/rand"
	"time"
)

type ImageTags struct {
	// The Semantic Version, i.e, v1.0.0
	Version string `json:"version"`
	// The SHA digest
	SHA string `json:"sha"`
}

type BlogCi struct{}

// Build container from Dockerfile
func (m *BlogCi) BuildFromDockerfile(
	// +defaultPath="/"
	source *dagger.Directory,
	platform dagger.Platform,
	tags ImageTags,
	// +default="http://localhost:8080/"
	base_url string,
) *dagger.Container {

	return source.DockerBuild(dagger.DirectoryDockerBuildOpts{
		Platform: platform,
		BuildArgs: []dagger.BuildArg{
			dagger.BuildArg{Name: "BASE_URL", Value: base_url},
			dagger.BuildArg{Name: "GIT_SHA", Value: tags.SHA},
			dagger.BuildArg{Name: "VERSION", Value: tags.Version},
		},
	}).WithLabel("org.opencontainers.image.created", time.Now().UTC().Format(time.RFC3339))
}

// Scan Image Built for Vunerabilities
func (m *BlogCi) ScanVunerabilities(ctx context.Context, ctr *dagger.Container) error {

	tarball := ctr.AsTarball()

	trivy := dag.Container().From("aquasec/trivy:0.68.2").
		WithMountedFile("/image.tar", tarball).
		WithExec([]string{
			"trivy",
			"image",
			"--input", "/image.tar",
			"--severity", "CRITICAL,HIGH",
			"--exit-code", "1", // signal critical/high vunerability found
			"--format", "table",
		})

	output, err := trivy.Stdout(ctx)
	if err != nil {
		return fmt.Errorf("critical/high vunerabilities detected: %s", err.Error())
	}

	fmt.Printf("Trivy scan success - no critical or high vunerabilites found\n%s", output)

	return nil
}

// Publish Docker image to registry
func (m *BlogCi) PublishImage(ctx context.Context, name string,
	// +default="latest"
	version string,
	sha string,
	// +default="ttl.sh"
	registry string,
	username string,
	password *dagger.Secret,
	// +optional
	base_url string,
	// +defaultPath="/"
	source *dagger.Directory,
) (string, error) {

	url := "http://localhost:8080/"
	if base_url != "" {
		url = base_url
	}

	platforms := []dagger.Platform{
		"linux/amd64",
		"linux/arm64",
	}
	platformVariants := make([]*dagger.Container, 0, len(platforms))
	for _, platform := range platforms {
		ctr := m.BuildFromDockerfile(source, platform, ImageTags{Version: version, SHA: sha}, url)
		if err := m.ScanVunerabilities(ctx, ctr); err != nil {
			return "", err
		}
		platformVariants = append(platformVariants, ctr)
	}

	imageName := fmt.Sprintf("%s/%s/%s:%s", registry, username, name, version)
	ctr := dag.Container()

	if registry != "ttl.sh" {
		ctr = ctr.WithRegistryAuth(registry, username, password)
	} else {
		imageName = fmt.Sprintf("%s/%s-%.0f", registry, name, math.Floor(rand.Float64()*10000000))
	}

	return ctr.Publish(ctx, imageName, dagger.ContainerPublishOpts{PlatformVariants: platformVariants})
}
