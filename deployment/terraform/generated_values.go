package terraform

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
)

const (
	// Name of the file where the generated values will be persisted.
	// The directory is always inferred from deployment.Config
	genValuesFileName = "generatedvalues.json"
)

// GeneratedValues is a struct containing values generated automatically during
// the deployment, not user-defined.
type GeneratedValues struct {
	GrafanaAdminPassword string `json:"GrafanaAdminPassword"`
}

func (v GeneratedValues) Sanitize() GeneratedValues {
	v.GrafanaAdminPassword = "********"
	return v
}

func getValuesPath(cfg deployment.Config) string {
	fileName := genValuesFileName
	return path.Join(cfg.TerraformStateDir, fileName)
}

func openValuesFile(cfg deployment.Config) (*os.File, error) {
	path := getValuesPath(cfg)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %q: %w", path, err)
	}

	return file, nil
}

func readGenValues(cfg deployment.Config) (*GeneratedValues, error) {
	file, err := openValuesFile(cfg)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read contents: %w", err)
	}

	if len(contents) == 0 {
		return &GeneratedValues{}, nil
	}

	var values GeneratedValues
	if err := json.Unmarshal(contents, &values); err != nil {
		return nil, fmt.Errorf("unable to unmarshal read content %q: %w", string(contents), err)
	}

	return &values, nil
}

func persistGeneratedValues(cfg deployment.Config, genValues *GeneratedValues) error {
	file, err := openValuesFile(cfg)
	if err != nil {
		return fmt.Errorf("unable to open file: %w", err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	if err := enc.Encode(genValues); err != nil {
		return fmt.Errorf("unable to encode generated values: %w", err)
	}

	return nil
}
