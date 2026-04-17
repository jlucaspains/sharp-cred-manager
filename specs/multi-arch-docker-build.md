# Multi-Architecture Docker Build (linux/amd64 + linux/arm64)

## Problem Statement
The current `deploy.yml` workflow builds and pushes a single `linux/amd64` Docker image. Users running on ARM64 hosts (e.g., Raspberry Pi, AWS Graviton, Apple Silicon via Docker) must rely on emulation. We want to publish a proper multi-arch manifest so Docker automatically pulls the correct image for the host architecture.

## Key Decisions
- **Runner strategy**: Use native GitHub-hosted runners (`ubuntu-latest` for amd64, `ubuntu-24.04-arm` for arm64) instead of QEMU emulation. Both are free for public repositories.
- **Publish strategy**: Each platform job pushes its image by digest (no tag). A final merge job combines the digests into a single multi-arch manifest under the existing tags (`latest`, semver).
- **Digest passing**: Use `actions/upload-artifact` / `actions/download-artifact` to pass per-platform digests from build jobs to the merge job.
- **Action versions**: Bump all Docker and checkout actions to their latest versions while touching the file.
- **Docker Hub description**: Keep the description update step in the merge job only (runs once, after both images are published).
- **No Dockerfile changes**: The existing multi-stage Dockerfile works as-is. Go produces static binaries and the `scratch`-based final image is architecture-agnostic from a Dockerfile perspective.

## Architecture Overview

### New Workflow Structure
```
deploy.yml (trigger: release created)
│
├── job: build (matrix)
│   ├── matrix[0]: runner=ubuntu-latest,    platform=linux/amd64
│   └── matrix[1]: runner=ubuntu-24.04-arm, platform=linux/arm64
│   │
│   └── steps:
│       - checkout
│       - docker/login-action
│       - docker/metadata-action        (image name only, no tags — tags set in merge)
│       - docker/setup-buildx-action
│       - docker/build-push-action      (push-by-digest=true, no tag)
│       - upload digest as artifact
│
└── job: merge (needs: build)
    └── steps:
        - download all digest artifacts
        - docker/login-action
        - docker/metadata-action        (produces final semver + latest tags)
        - docker/buildx-imagetools      (create manifest from digests + attach tags)
        - peter-evans/dockerhub-description
```

### Digest Artifact Convention
Each build job exports the image digest and uploads it as an artifact named `digest-<sanitized-platform>` (e.g., `digest-linux-amd64`, `digest-linux-arm64`). The merge job downloads all artifacts matching `digest-*` and constructs the `imagetools create` command.

---

## Tasks

### 1. Update deploy.yml – Build Job (Matrix)
> Status: Complete

- Replace the single `backend` job with a `build` job using a matrix strategy:
  ```yaml
  strategy:
    fail-fast: false
    matrix:
      include:
        - platform: linux/amd64
          runner: ubuntu-latest
        - platform: linux/arm64
          runner: ubuntu-24.04-arm
  ```
- Set `runs-on: ${{ matrix.runner }}`
- Use `docker/metadata-action@v6` to produce the image name (no tags needed at this stage)
- Use `docker/setup-buildx-action@v4`
- Use `docker/build-push-action@v7` with:
  - `platforms: ${{ matrix.platform }}`
  - `outputs: type=image,push-by-digest=true,name-canonical=true,push=true`
  - Capture the digest from the step output (`steps.build.outputs.digest`)
- Upload the digest as an artifact (`actions/upload-artifact@v7`), one file per platform

### 2. Update deploy.yml – Merge Job
> Status: Complete

- Add a `merge` job with `needs: build` running on `ubuntu-latest`
- Download all digest artifacts (`actions/download-artifact@v8` with `pattern: digest-*`, `merge-multiple: true`)
- Use `docker/metadata-action@v6` to produce final semver + latest tags (same config as today)
- Run `docker buildx imagetools create` as a shell step to create the multi-arch manifest:
  - Buildx is already available after `docker/setup-buildx-action`
  - Input: all downloaded digests
  - Tags: from metadata-action output
- Move the `peter-evans/dockerhub-description@v5` step into this job

### 3. Bump Action Versions
> Status: Complete

- `actions/checkout@v3` → `actions/checkout@v4`
- `docker/metadata-action@v4` → `docker/metadata-action@v6`
- `docker/login-action@v2` → `docker/login-action@v4`
- `docker/setup-buildx-action@v2` → `docker/setup-buildx-action@v4`
- `docker/build-push-action@v4` → `docker/build-push-action@v7`
- `actions/upload-artifact` (new) → `v7`
- `actions/download-artifact` (new) → `v8`
