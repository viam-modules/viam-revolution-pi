package revolution_pi

import (
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/utils"
)

// Model is the model triplet for the rev-pi board.
var Model = resource.NewModel("viam-labs", "kunbus", "revolutionpi")

// Config is the config for the rev-pi board.
type Config struct {
	resource.TriviallyValidateConfig
	Attributes utils.AttributeMap `json:"attributes,omitempty"`
}
