# viam-revolution-pi

A modular component for Viam that adds support for the Revolution Pi PLC platform

## Setup and Configuration

Please follow the [Revolution Pi](https://revolutionpi.com/en/tutorials/quick-start-guide) setup documentation to configure your Revolution Pi. The majority of the configuration for a Revolution Pi occurs within [PiCtory](https://revolutionpi.com/en/tutorials/what-is-pictory).

The `board` component does not require any additional configuration.

### GPIO and PWM

The family of boards used for digital input and output are the [DIO modules](https://revolutionpi.com/en/tutorials/overview-revpi-io-modules). These have a set of GPIO pins to use with PWMs and counters. To configure an Output pin as a PWM pin, you must set the corresponding bit for that pin in the 'OutputPWMActive' Word in PiCtory. Because OutputPWMActive is stored in memory, you have to update the field in PiCtory, then update the Start-Config that the rev-pi uses and restart the board. The PWM frequency can also only be configured in PiCtory by updating the 'OutputPWMFrequency' field. Every PWM pin will use the same frequency.

Interrupts and counters are not currently supported on the board

#### example enabling a PWM pin

If you want to enable pins O_3 and O_9 as PWM pins, take the following steps

1.  First update the 'OutputPWMActive' field in PiCtory
    - The binary representation for enabling these two pins would be represented as 0b0000000100000100, with the decimal equivalent being 260
2.  Then you save your changes in PiCtory as the latest Start-Config
3.  Restart your Revolution Pi

This will enable pins O_3 and O_9 as PWM pins, which can be used with Viam's APIs. This also means that O_3 and O_9 can no longer be used as normal GPIO pins.

### ADC and DAC

The [AIO Module](https://revolutionpi.com/en/tutorials/overview-aio) is used for analog inputs and outputs on the Revolution Pi. The module currently supports 4 analog readers and 2 analog writers. the RTD analog readers are currently not managed by this module. See [RTD Measurement Documentation](https://revolutionpi.com/en/tutorials/overview-aio/rtd-measurement) for the Revolution Pi for more information.

### DoCommand

A DoCommand is configured to read from any address supported in the Revolution Pi. The command is configured as

```
{"readParameter": <PARAMETER_NAME>}
```

This is useful for reading values that would normally not be supported through the board APIs, such as checking `RevPiStatus` or `Core_Temperature`.
