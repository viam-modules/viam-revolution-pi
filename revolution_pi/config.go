package revolution_pi

import (
	"go.viam.com/rdk/utils"
)

type Config struct {
	Attributes utils.AttributeMap `json:"attributes,omitempty"`
}

// Validate ensures all parts of the config are valid.
func (conf *Config) Validate(path string) ([]string, error) {
	return nil, nil
}
