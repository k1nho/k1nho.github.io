package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"dagger/blog-ci/internal/dagger"
)

type ImageTags struct {
	// The Semantic Version, i.e, v1.0.0
	Version string
	// The SHA digest
	SHA string
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
	return dag.Container(dagger.ContainerOpts{Platform: platform}).
		Build(source, dagger.ContainerBuildOpts{
			BuildArgs: []dagger.BuildArg{
				dagger.BuildArg{Name: "BASE_URL", Value: base_url},
				dagger.BuildArg{Name: "GIT_SHA", Value: tags.SHA},
				dagger.BuildArg{Name: "VERSION", Value: tags.Version},
			},
		}).
		WithLabel("org.opencontainers.image.created", time.Now().UTC().Format(time.RFC3339))
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
	// +defaultPath="/"
	source *dagger.Directory,
) (string, error) {

	platforms := []dagger.Platform{
		"linux/amd64",
		"linux/arm64",
	}
	platformVariants := make([]*dagger.Container, 0, len(platforms))
	for _, platform := range platforms {
		platformVariants = append(platformVariants, m.BuildFromDockerfile(source, platform, ImageTags{Version: version, SHA: sha}, "http://localhost:8080/"))
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
