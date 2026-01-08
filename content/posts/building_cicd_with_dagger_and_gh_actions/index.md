---
title: "Building a CI/CD Pipeline for Container Images with Dagger and GitHub Actions"
pubDate: 2026-01-20
Description: "There comes a time where a container image is needed, and with that a pipeline. Let's build our own with Dagger!"
Categories: ["CI/CD", "Platform Engineering", "Containers"]
Tags: ["CI/CD", "DevOps", "Containers"]
cover: "gallery/homelabs_cover3.png"
images:
  - "allery/homelab_in3.png"
mermaid: true
draft: true
---

Recently, I've been working on my homelab, and one thing that eventually becomes essential is having **reliable pipelines for building, scanning, and publishing container images**.
Whether you're developing custom applications from scratch or creating hardened versions of existing images to strengthen their security posture, you need a robust build process you can trust.

While there are plenty of GitHub Actions in the marketplace that can handle this, there's something uniquely powerful about owning a custom pipeline where you control every piece without drowning in YAML hell. Even better, what if you could define your entire pipeline in real code and test it locally without waiting for CI runners? Enter [Dagger](https://dagger.io/). With Dagger, your local pipeline runs are identical to what executes in CI, giving you fast feedback loops and the confidence that what works on your machine will work in production.

In this guide, I'll walk you through setting up a complete pipeline that builds a container image, scans it for security vulnerabilities, and publishes it to a registry of your choice, all using Go!

## The Stage

I wanted to have a container image for the blog that will be deployed on my tailnet via Argo CD. The app here is a simple static Hugo blog running on an nginx base.

## Daggerizing the Pipeline

### Initializing Dagger Module

Let's start by initializing our dagger module this is where our CI code will live.

```bash
dagger init --sdk=go --name=blog-ci
```

The following will create the `.dagger` directory with some boiler plate, I choose [Go](https://go.dev/) for the sdk, but feel free
to choose any language from their [available sdks](https://docs.dagger.io/getting-started/api/sdk/).

### Build CI

In order to have a container image to use for our Kubernetes deployment, we need to have a pipeline
that will build and publish the container image. Let's start with the build.

```go
type ImageTags struct {
	// The Semantic Version, i.e, v1.0.0
	Version string `json:"version"`
	// The SHA digest
	SHA string `json:"sha"`
}
```

```go {filename=".dagger/main.go"}
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
```

We define the `BuildFromDockerfile` that takes some parameters:

- `source`: the directory where the Dockerfile is located
- `platform`: the specific platform variant, i.e, linux/amd64
- `tags`: A struct that contains both semantic version and github sha
- `base_url`: Hugo specific config for the base url of the website

This will return a dagger container correctly tagged and ready to be published to a container registry.

### Security Scanning with Trivy

```go
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

```

### Publish CI

Now, that we have a dagger function that will built a container from us, we can publish that container
by providing the specified parameters.

```go {filename=".dagger/main.go", linenos=true, hl_lines=["38-42"]}
// Publish Docker image to registry
func (m *BlogCi) PublishImage(ctx context.Context,
	name string,
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
```

The logic from line 38-42 help us test the pipeline locally by publishing the container image to [ttl.sh](https://ttl.sh/),
and if we provide a different registry such as **ghcr.io** or **dockerhub**, it will apply the registry auth.

### Finishing the pipeline with Github Actions

Lastly, we can wrap our dagger call and have a simple github action that will run when we publish a tag as follows:

```yaml {filename=".github/workflows/publish.yaml"}
name: Publish Blog Image

on:
  push:
    tags:
      - "**"
jobs:
  publish:
    runs-on: ubuntu-24.04
    permissions:
      contents: read
      packages: write
    env:
      NAME: kinho-blog
      USERNAME: ${{github.repository_owner}}
      SHA_TAG: ${{github.sha}}
      SEMVER_TAG: ${{github.ref_name}}

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          submodules: true
          fetch-depth: 0
      - uses: dagger/dagger-for-github@8.0.0

      - name: Publish Blog Docker Image to ghcr
        env:
          PASSWORD: ${{ secrets.GITHUB_TOKEN }}
        run: |
          dagger call publish-image --registry=ghcr.io --name=$NAME --version=latest --sha=$SHA_TAG --username=$USERNAME --password=env:PASSWORD # latest
          dagger call publish-image --registry=ghcr.io --name=$NAME --version=$SEMVER_TAG --sha=$SHA_TAG --username=$USERNAME --password=env:PASSWORD # semver
          dagger call publish-image --registry=ghcr.io --name=$NAME --version=$SHA_TAG --sha=$SHA_TAG --username=$USERNAME --password=env:PASSWORD # sha
```
