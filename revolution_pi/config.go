package revolution_pi

import (
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/utils"
)

var Model = resource.NewModel("viam-labs", "kunbus", "revolutionpi")

type Config struct {
	resource.TriviallyValidateConfig
	Attributes utils.AttributeMap `json:"attributes,omitempty"`
}
