// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shamir

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	mathrand "math/rand"
	"time"
)

const (
	ShareOverhead = 1
)

type polynomial struct {
	coefficients []uint8
}

func makePolynomial(intercept, degree uint8) (polynomial, error) {
	p := polynomial{
		coefficients: make([]byte, degree+1),
	}
	p.coefficients[0] = intercept
	if _, err := rand.Read(p.coefficients[1:]); err != nil {
		return p, err
	}
	return p, nil
}

func (p *polynomial) evaluate(x uint8) uint8 {
	if x == 0 {
		return p.coefficients[0]
	}
	degree := len(p.coefficients) - 1
	out := p.coefficients[degree]
	for i := degree - 1; i >= 0; i-- {
		coeff := p.coefficients[i]
		out = add(mult(out, x), coeff)
	}
	return out
}

func interpolatePolynomial(x_samples, y_samples []uint8, x uint8) uint8 {
	limit := len(x_samples)
	var result, basis uint8
	for i := 0; i < limit; i++ {
		basis = 1
		for j := 0; j < limit; j++ {
			if i == j {
				continue
			}
			num := add(x, x_samples[j])
			denom := add(x_samples[i], x_samples[j])
			term := div(num, denom)
			basis = mult(basis, term)
		}
		group := mult(y_samples[i], basis)
		result = add(result, group)
	}
	return result
}

func div(a, b uint8) uint8 {
	if b == 0 {
		panic("divide by zero")
	}
	ret := int(mult(a, inverse(b)))
	ret = subtle.ConstantTimeSelect(subtle.ConstantTimeByteEq(a, 0), 0, ret)
	return uint8(ret)
}

func inverse(a uint8) uint8 {
	b := mult(a, a)
	c := mult(a, b)
	b = mult(c, c)
	b = mult(b, b)
	c = mult(b, c)
	b = mult(b, b)
	b = mult(b, b)
	b = mult(b, c)
	b = mult(b, b)
	b = mult(a, b)
	return mult(b, b)
}

func mult(a, b uint8) (out uint8) {
	var r uint8 = 0
	var i uint8 = 8
	for i > 0 {
		i--
		r = (-(b >> i & 1) & a) ^ (-(r >> 7) & 0x1B) ^ (r + r)
	}
	return r
}

func add(a, b uint8) uint8 {
	return a ^ b
}

// Split takes an arbitrarily long secret and generates a `parts`
// number of shares, `threshold` of which are required to reconstruct
// the secret. The parts and threshold must be at least 2, and less
// than 256. The returned shares are each one byte longer than the secret
// as they attach a tag used to reconstruct the secret.
func Split(secret []byte, parts, threshold int) ([][]byte, error) {
	if parts < threshold {
		return nil, fmt.Errorf("parts cannot be less than threshold")
	}
	if parts > 255 {
		return nil, fmt.Errorf("parts cannot exceed 255")
	}
	if threshold < 2 {
		return nil, fmt.Errorf("threshold must be at least 2")
	}
	if threshold > 255 {
		return nil, fmt.Errorf("threshold cannot exceed 255")
	}
	if len(secret) == 0 {
		return nil, fmt.Errorf("cannot split an empty secret")
	}

	mathrand.Seed(time.Now().UnixNano())
	xCoordinates := mathrand.Perm(255)

	out := make([][]byte, parts)
	for idx := range out {
		out[idx] = make([]byte, len(secret)+1)
		out[idx][len(secret)] = uint8(xCoordinates[idx]) + 1
	}

	for idx, val := range secret {
		p, err := makePolynomial(val, uint8(threshold-1))
		if err != nil {
			return nil, fmt.Errorf("failed to generate polynomial: %w", err)
		}
		for i := 0; i < parts; i++ {
			x := uint8(xCoordinates[i]) + 1
			y := p.evaluate(x)
			out[i][idx] = y
		}
	}

	return out, nil
}

// Combine is used to reverse a Split and reconstruct a secret
// once a `threshold` number of parts are available.
func Combine(parts [][]byte) ([]byte, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("less than two parts cannot be used to reconstruct the secret")
	}

	firstPartLen := len(parts[0])
	if firstPartLen < 2 {
		return nil, fmt.Errorf("parts must be at least two bytes")
	}
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) != firstPartLen {
			return nil, fmt.Errorf("all parts must be the same length")
		}
	}

	secret := make([]byte, firstPartLen-1)
	x_samples := make([]uint8, len(parts))
	y_samples := make([]uint8, len(parts))

	checkMap := map[byte]bool{}
	for i, part := range parts {
		samp := part[firstPartLen-1]
		if exists := checkMap[samp]; exists {
			return nil, fmt.Errorf("duplicate part detected")
		}
		checkMap[samp] = true
		x_samples[i] = samp
	}

	for idx := range secret {
		for i, part := range parts {
			y_samples[i] = part[idx]
		}
		val := interpolatePolynomial(x_samples, y_samples, 0)
		secret[idx] = val
	}
	return secret, nil
}
