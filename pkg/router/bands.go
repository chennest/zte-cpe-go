package router

import "fmt"

// LteBand represents an LTE band.
type LteBand int

const (
	// Band1: 2100 MHz (IMT)
	Band1 LteBand = 1
	// Band3: 1800 MHz (DCS)
	Band3 LteBand = 3
	// Band7: 2600 MHz (IMT-E)
	Band7 LteBand = 7
	// Band8: 900 MHz (Extended GSM)
	Band8 LteBand = 8
	// Band20: 800 MHz (Digital Dividend, EU)
	Band20 LteBand = 20
	// Band28: 700 MHz (APT)
	Band28 LteBand = 28
	// Band32: 1500 MHz (L-Band, SDL)
	Band32 LteBand = 32
	// Band38: 2600 MHz (TDD, IMT-E)
	Band38 LteBand = 38
)

// AllLTEBands is the predefined bitmask representing all supported LTE bands.
const AllLTEBands = "0x20080800C5"

// SelectLTEBand converts selected bands to a hexadecimal bitmask.
// If bands is nil, returns AllLTEBands.
func SelectLTEBand(bands []LteBand) string {
	if bands == nil {
		return AllLTEBands
	}
	return calculateBitmask(bands)
}

func calculateBitmask(bands []LteBand) string {
	var bitmask uint64
	for _, band := range bands {
		bitmask |= 1 << (uint64(band) - 1)
	}
	return fmt.Sprintf("0x%X", bitmask)
}

// ParseLTEBand parses a string into an LteBand.
func ParseLTEBand(s string) (LteBand, error) {
	switch s {
	case "1", "band1", "Band1":
		return Band1, nil
	case "3", "band3", "Band3":
		return Band3, nil
	case "7", "band7", "Band7":
		return Band7, nil
	case "8", "band8", "Band8":
		return Band8, nil
	case "20", "band20", "Band20":
		return Band20, nil
	case "28", "band28", "Band28":
		return Band28, nil
	case "32", "band32", "Band32":
		return Band32, nil
	case "38", "band38", "Band38":
		return Band38, nil
	default:
		return 0, fmt.Errorf("unsupported LTE band: %s", s)
	}
}
