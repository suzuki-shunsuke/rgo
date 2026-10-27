# Install

rgo is written in Go. So you only have to install a binary in your `PATH`.

There are some ways to install rgo.

1. [Homebrew](#homebrew)
1. [Scoop](#scoop)
1. [aqua](#aqua)
1. [GitHub Releases](#github-releases)
1. [Build an executable binary from source code yourself using Go](#build-an-executable-binary-from-source-code-yourself-using-go)

## Homebrew

You can install rgo using [Homebrew](https://brew.sh/).

```sh
brew install suzuki-shunsuke/rgo/rgo
```

## Scoop

You can install rgo using [Scoop](https://scoop.sh/).

```sh
scoop bucket add suzuki-shunsuke https://github.com/suzuki-shunsuke/scoop-bucket
scoop install rgo
```

## aqua

You can install rgo using [aqua](https://aquaproj.github.io/).

[aqua-registry >= v4.448.0](https://github.com/aquaproj/aqua-registry/releases/tag/v4.448.0)

```sh
aqua g -i suzuki-shunsuke/rgo
```

## Build an executable binary from source code yourself using Go

```sh
go install github.com/suzuki-shunsuke/rgo/cmd/rgo@latest
```

## GitHub Releases

You can download an asset from [GitHub Releases](https://github.com/suzuki-shunsuke/rgo/releases).
Please unarchive it and install a pre built binary into `$PATH`. 

### Verify downloaded assets from GitHub Releases

You can verify downloaded assets using some tools.

1. [GitHub CLI](https://cli.github.com/)
1. [Cosign](https://github.com/sigstore/cosign)
1. [Packslip](https://packslip.dev/) `rgo >= v0.1.1`
1. [slsa-verifier](https://github.com/slsa-framework/slsa-verifier) `rgo <= v0.1.0`

### 1. GitHub CLI

You can install GitHub CLI by aqua.

```sh
aqua g -i cli/cli
```

```sh
version=v0.1.2
asset=rgo_darwin_arm64.tar.gz
gh release download -R suzuki-shunsuke/rgo "$version" -p "$asset"
gh attestation verify "$asset" \
  -R suzuki-shunsuke/rgo \
  --signer-workflow suzuki-shunsuke/go-release-workflow/.github/workflows/release.yaml
```

### 2. Cosign

You can install Cosign by aqua.

```sh
aqua g -i sigstore/cosign
```

From rgo v0.1.1, the bundle file is `rgo_<version>_checksums.txt.sigstore.json`.
Cosign v2.4.2 or later is required.

```sh
version=v0.1.2
checksum_file="rgo_${version#v}_checksums.txt"
asset=rgo_darwin_arm64.tar.gz
gh release download "$version" \
  -R suzuki-shunsuke/rgo \
  -p "$asset" \
  -p "$checksum_file" \
  -p "${checksum_file}.sigstore.json"
cosign verify-blob \
  "$checksum_file" \
  --bundle "${checksum_file}.sigstore.json" \
  --certificate-identity-regexp 'https://github\.com/suzuki-shunsuke/go-release-workflow/\.github/workflows/release\.yaml@.*' \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"
cat "$checksum_file" | sha256sum -c --ignore-missing -
```

For rgo v0.1.0, `*.pem` and `*.sig` are available instead.

```sh
version=v0.1.0
checksum_file="rgo_${version#v}_checksums.txt"
asset=rgo_darwin_arm64.tar.gz
gh release download "$version" \
  -R suzuki-shunsuke/rgo \
  -p "$asset" \
  -p "$checksum_file" \
  -p "${checksum_file}.pem" \
  -p "${checksum_file}.sig"
cosign verify-blob \
  --signature "${checksum_file}.sig" \
  --certificate "${checksum_file}.pem" \
  --certificate-identity-regexp 'https://github\.com/suzuki-shunsuke/go-release-workflow/\.github/workflows/release\.yaml@.*' \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  "$checksum_file"
cat "$checksum_file" | sha256sum -c --ignore-missing -
```

### 3. Packslip

`rgo >= v0.1.1`

You can install packslip by aqua.

```sh
aqua g -i jdx/packslip
```

```sh
version=v0.1.2
asset=rgo_darwin_arm64.tar.gz
gh release download -R suzuki-shunsuke/rgo "$version" -p "$asset" -p packslip.sigstore.json
packslip verify packslip.sigstore.json \
    --pin ps1_6ndmpdmfsikfiawshynnur5qky \
    --identity-prefix https://github.com/suzuki-shunsuke/go-release-workflow/.github/workflows/release.yaml@ \
    --artifact "$asset"
```

- `--pin` requires the release to be signed from the repository `suzuki-shunsuke/rgo`, identified by its repository ID rather than its name
- `--identity-prefix` requires the release to be signed by the release workflow of [suzuki-shunsuke/go-release-workflow](https://github.com/suzuki-shunsuke/go-release-workflow)

On success, packslip prints a line starting with `ok:`.
Please make sure it shows the project and version you meant, e.g. `github.com/suzuki-shunsuke/rgo 0.1.2`.

### 4. slsa-verifier

> [!WARNING]
> As of rgo v0.1.1, `multiple.intoto.jsonl` hasn't been published anymore.

You can install slsa-verifier by aqua.

```sh
aqua g -i slsa-framework/slsa-verifier
```

```sh
version=v0.1.0
asset=rgo_darwin_arm64.tar.gz
gh release download -R suzuki-shunsuke/rgo "$version" -p "$asset" -p multiple.intoto.jsonl
slsa-verifier verify-artifact "$asset" \
  --provenance-path multiple.intoto.jsonl \
  --source-uri github.com/suzuki-shunsuke/rgo \
  --source-tag "$version"
```
