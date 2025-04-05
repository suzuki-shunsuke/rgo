# rgo

[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/suzuki-shunsuke/rgo/main/LICENSE) | [script](rgo)

rgo is a tiny script to release a Homebrew-tap recipe, Scoop App Manifest, and a winget manifest built with GoReleaser

:warning: Winget isn't support yet.

## Requirements

- Bash
- Git
- GitHub CLI

## How To Install

Please copy [rgo](rgo) into `$PATH`.

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
rgo "<released version>"
```

e.g.

```sh
rgo v0.1.0
```
