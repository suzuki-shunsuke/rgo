# rgo

[![DeepWiki](https://img.shields.io/badge/DeepWiki-suzuki--shunsuke%2Frgo-blue.svg?logo=data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACwAAAAyCAYAAAAnWDnqAAAAAXNSR0IArs4c6QAAA05JREFUaEPtmUtyEzEQhtWTQyQLHNak2AB7ZnyXZMEjXMGeK/AIi+QuHrMnbChYY7MIh8g01fJoopFb0uhhEqqcbWTp06/uv1saEDv4O3n3dV60RfP947Mm9/SQc0ICFQgzfc4CYZoTPAswgSJCCUJUnAAoRHOAUOcATwbmVLWdGoH//PB8mnKqScAhsD0kYP3j/Yt5LPQe2KvcXmGvRHcDnpxfL2zOYJ1mFwrryWTz0advv1Ut4CJgf5uhDuDj5eUcAUoahrdY/56ebRWeraTjMt/00Sh3UDtjgHtQNHwcRGOC98BJEAEymycmYcWwOprTgcB6VZ5JK5TAJ+fXGLBm3FDAmn6oPPjR4rKCAoJCal2eAiQp2x0vxTPB3ALO2CRkwmDy5WohzBDwSEFKRwPbknEggCPB/imwrycgxX2NzoMCHhPkDwqYMr9tRcP5qNrMZHkVnOjRMWwLCcr8ohBVb1OMjxLwGCvjTikrsBOiA6fNyCrm8V1rP93iVPpwaE+gO0SsWmPiXB+jikdf6SizrT5qKasx5j8ABbHpFTx+vFXp9EnYQmLx02h1QTTrl6eDqxLnGjporxl3NL3agEvXdT0WmEost648sQOYAeJS9Q7bfUVoMGnjo4AZdUMQku50McDcMWcBPvr0SzbTAFDfvJqwLzgxwATnCgnp4wDl6Aa+Ax283gghmj+vj7feE2KBBRMW3FzOpLOADl0Isb5587h/U4gGvkt5v60Z1VLG8BhYjbzRwyQZemwAd6cCR5/XFWLYZRIMpX39AR0tjaGGiGzLVyhse5C9RKC6ai42ppWPKiBagOvaYk8lO7DajerabOZP46Lby5wKjw1HCRx7p9sVMOWGzb/vA1hwiWc6jm3MvQDTogQkiqIhJV0nBQBTU+3okKCFDy9WwferkHjtxib7t3xIUQtHxnIwtx4mpg26/HfwVNVDb4oI9RHmx5WGelRVlrtiw43zboCLaxv46AZeB3IlTkwouebTr1y2NjSpHz68WNFjHvupy3q8TFn3Hos2IAk4Ju5dCo8B3wP7VPr/FGaKiG+T+v+TQqIrOqMTL1VdWV1DdmcbO8KXBz6esmYWYKPwDL5b5FA1a0hwapHiom0r/cKaoqr+27/XcrS5UwSMbQAAAABJRU5ErkJggg==)](https://deepwiki.com/suzuki-shunsuke/rgo)
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/suzuki-shunsuke/rgo/main/LICENSE) | [INSTALL](INSTALL.md)

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
