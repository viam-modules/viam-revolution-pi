//go:build linux

// Package revolution_pi implements the Revolution Pi board GPIO pins.
package revolution_pi

import "context"

type revolutionPiGpioPin struct {
}

func (pin *revolutionPiGpioPin) Set(ctx context.Context, high bool, extra map[string]interface{}) error {
	return nil
}

// Get gets the high/low state of the pin.
func (pin *revolutionPiGpioPin) Get(ctx context.Context, extra map[string]interface{}) (bool, error) {
	return false, nil
}

// PWM gets the pin's given duty cycle.
func (pin *revolutionPiGpioPin) PWM(ctx context.Context, extra map[string]interface{}) (float64, error) {
	return 0, nil
}

// SetPWM sets the pin to the given duty cycle.
func (pin *revolutionPiGpioPin) SetPWM(ctx context.Context, dutyCyclePct float64, extra map[string]interface{}) error {
	return nil
}

// PWMFreq gets the PWM frequency of the pin.
func (pin *revolutionPiGpioPin) PWMFreq(ctx context.Context, extra map[string]interface{}) (uint, error) {
	return 0, nil
}

// SetPWMFreq sets the given pin to the given PWM frequency. For Raspberry Pis,
// 0 will use a default PWM frequency of 800.
func (pin *revolutionPiGpioPin) SetPWMFreq(ctx context.Context, freqHz uint, extra map[string]interface{}) error {
	return nil
}
