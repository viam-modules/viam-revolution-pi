// Package revolutionpi implements the Revolution Pi board GPIO pins.
package revolutionpi

func str32(chars [32]byte) string {
	i := 0
	var c byte
	for i, c = range chars {
		if c == 0 {
			break
		}
	}
	return string(chars[:i])
}

func char32(str string) (chars [32]byte) {
	copy(chars[:31], str)
	return
}
