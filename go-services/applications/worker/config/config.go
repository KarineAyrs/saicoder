package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/KarineAyrs/safe-ai-coder/pkg/config"
	"gopkg.in/yaml.v2"
)

type Worker struct {
	API          config.API         `yaml:"api"`
	Ops          config.Ops         `yaml:"ops"`
	SchedulerAPI config.ExternalAPI `yaml:"scheduler_api"`
	Coder        config.Coder       `yaml:"coder"`
}

func Parse(filePath string) (Worker, error) {
	c := Worker{}

	f, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return c, fmt.Errorf("cannot read from file: %s, err: %w", filePath, err)
	}

	d := yaml.NewDecoder(f)
	// Because there are multiple consumers of the configuration being parsed here, it is much simpler to ignore extra fields
	// compared to adjusting configuration source (Helm, inventory folder) to contain only those fields we've defined in this code.
	d.SetStrict(false)

	err = d.Decode(&c)
	if err != nil {
		return c, fmt.Errorf("error decoding file: %s, err: %w", filePath, err)
	}

	err = f.Close()
	if err != nil {
		return c, fmt.Errorf("cannot close file: %s, err: %w", filePath, err)
	}

	return c, nil
}
