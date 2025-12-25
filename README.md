# rgo

[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/suzuki-shunsuke/rgo/main/LICENSE)

rgo is a CLI to release a Homebrew-tap recipe, Scoop App Manifest, and a winget manifest built with GoReleaser.

## Why is rgo necessary?

To release Homebrew-tap recipe, Scoop App Manifest, and a winget manifest built with GoReleaser without passing a GitHub Personal Access Token or GitHub App to CI.

When we build and release Go Application using GoReleaser in CI, we usually release files such as Homebrew-tap recipe in CI too.
But these files are hosted in different repositories from Go code, so GitHub Actions token isn't available to release them.
Either a personal access token or GitHub App is required, which is undesirable in terms of security because these secrets may be leaked and abused.

rgo resolves this issue.
CI builds files such as Homebrew-tap recipe and uploads them to GitHub Actions Artifacts, and rgo downloads and releases them from out of CI (probably your machine).
Then you don't need to pass secrets to CI.

The drawback of rgo is that rgo depends on the environment out of CI (probably your machine).
But released files are built in CI.
rgo only downloads and releases them.
So we think we can accept the drawback.

## Requirements

- Git
- GitHub CLI

## How does it work?

rgo does the following things:

1. Create and push a given tag
2. Wait until the release workflow completes
3. Create a temporary directory to work on
4. Downloads files from GitHub Actions Artifacts
5. Checkout repositories (`homebrew-*`, `scoop-bucket`, and `winget-pkgs`)
6. Push Homebrew-tap recipe and Scoop App Manifest
7. Create a pull request to winget-pkgs

## How To Use

1. Edit .goreleaser.yml:

`skip_upload: true`

2. Edit the release workflow to upload files to GitHub Actions Artifacts:

```yaml
- name: Run GoReleaser
  run: goreleaser release --clean
  env:
    GITHUB_TOKEN: ${{ github.token }}
- uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02 # v4.6.2
  if: "!contains(github.ref_name, '-')" # Skip on prerelease
  with:
    name: goreleaser
    path: |
      dist/homebrew/*.rb
      dist/scoop/*.json
```

3. Run `rgo` on the released repository:

```sh
rgo run "<released version>"
```

e.g.

```sh
rgo run v0.1.0
```
