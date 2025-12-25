package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ProjectName   string         `yaml:"project_name"`
	HomebrewCasks []HomebrewCask `yaml:"homebrew_casks"`
	Brews         []Brew         `yaml:"brews"`
	Scoops        []Scoop        `yaml:"scoops"`
	Winget        []Winget       `yaml:"winget"`
}

type HomebrewCask struct {
	Repository Repository `yaml:"repository"`
}

type Brew struct {
	Repository Repository `yaml:"repository"`
}

type Scoop struct {
	Repository Repository `yaml:"repository"`
}

type Winget struct {
	Publisher  string     `yaml:"publisher"`
	Repository WingetRepo `yaml:"repository"`
}

type Repository struct {
	Owner  string `yaml:"owner"`
	Name   string `yaml:"name"`
	Branch string `yaml:"branch"`
}

type WingetRepo struct {
	Owner       string      `yaml:"owner"`
	Name        string      `yaml:"name"`
	Branch      string      `yaml:"branch"`
	PullRequest PullRequest `yaml:"pull_request"`
}

type PullRequest struct {
	Enabled bool       `yaml:"enabled"`
	Draft   bool       `yaml:"draft"`
	Base    Repository `yaml:"base"`
}

func Read(fs afero.Fs, cfgFilePath string) (*Config, error) {
	if cfgFilePath != "" {
		return readFile(fs, cfgFilePath)
	}
	cfg, err := readFile(fs, ".goreleaser.yaml")
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return readFile(fs, ".goreleaser.yml")
}

func readFile(fs afero.Fs, p string) (*Config, error) {
	f, err := fs.Open(p)
	if err != nil {
		return nil, fmt.Errorf("open a config file: %w", err)
	}
	defer f.Close() //nolint:errcheck
	cfg := &Config{}
	if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
		return nil, fmt.Errorf("decode a config file as YAML: %w", err)
	}
	return cfg, nil
}
