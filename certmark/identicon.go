package certmark

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/rrivera/identicon"
)

var (
	DefaultIdenticonConfig = GraphIdenticonConfig{
		BlockSize: 7,
		Colour:    "yellow",
		Density:   3,
	}
)

type GraphIdenticonConfig struct {
	BlockSize int
	Colour    string
	Density   int
}

func GenerateIdenticon(inputBytes []byte, config GraphIdenticonConfig) ([]byte, error) {
	alwaysRed := func(cb []byte) color.Color {
		return color.RGBA{255, 0, 0, 255}
	}

	transparentBg := func(cb []byte, fc color.Color) color.Color {
		return color.Transparent
	}

	gen, err := identicon.New(
		"certmark",
		config.BlockSize,
		config.Density,
		identicon.SetRandom(false),
		identicon.SetFillColorFunction(alwaysRed),
		identicon.SetBackgroundColorFunction(transparentBg),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise identicon: %w", err)
	}

	ii, err := gen.Draw(string(inputBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to draw identicon: %w", err)
	}

	asciiCanvas := asciiCanvas(&ii.Canvas)

	return joinArray(asciiCanvas, " ", " "), nil
}

func asciiCanvas(c *identicon.Canvas) [][]string {
	input := c.Array()
	log.Debugf("input:\n%d", input)

	height := len(input)
	width := len(input[0])
	output := make([][]string, height)

	for i := range output {
		output[i] = make([]string, width)
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := input[y][x]
			switch {
			case value == 0:
				output[y][x] = " "
			case value >= 1 && value <= 3:
				output[y][x] = "░"
			case value >= 4 && value <= 6:
				output[y][x] = "▒"
			case value >= 7 && value <= 9:
				output[y][x] = "▓"
			default:
				output[y][x] = "▓"
			}
		}
	}

	log.Debugf("asciiCanvas\n%s", output)
	return output
}

func joinArray(input [][]string, separator string, fillEmpty string) []byte {
	height := len(input)
	width := len(input[0])

	var output strings.Builder
	output.Grow(height*(width+len(separator)) + height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := input[y][x]
			if c == " " {
				output.WriteString(fillEmpty)
			} else {
				output.WriteString(input[y][x])
			}
			if x < width-1 {
				output.WriteString(separator)
			}
		}
		if y < height-1 {
			output.WriteString("\n")
		}
	}

	return []byte(output.String())
}
