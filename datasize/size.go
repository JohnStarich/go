// Package datasize parses, formats, and converts to different units in bytes.
package datasize

import (
	"fmt"
	"math/big"
)

//nolint:gochecknoglobals,mnd // These are all effectively constants as big.Int types.
var (
	kilobytes  = big.NewInt(1e3)
	megabytes  = big.NewInt(1e6)
	gigabytes  = big.NewInt(1e9)
	terabytes  = big.NewInt(1e12)
	petabytes  = big.NewInt(1e15)
	exabytes   = big.NewInt(1e18)
	zettabytes = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e3))
	yottabytes = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6))

	kibibytes = big.NewInt(1 << 10)
	mebibytes = big.NewInt(1 << 20)
	gibibytes = big.NewInt(1 << 30)
	tebibytes = big.NewInt(1 << 40)
	pebibytes = big.NewInt(1 << 50)
	exbibytes = big.NewInt(1 << 60)
	zebibytes = new(big.Int).Mul(big.NewInt(1<<60), big.NewInt(1<<10))
	yobibytes = new(big.Int).Mul(big.NewInt(1<<60), big.NewInt(1<<20))
)

// Size represents a quantity of bytes
type Size struct {
	value *big.Int
}

// New returns a Size for the given number of bytes
func New(bytes int64) Size {
	return NewFromInt(big.NewInt(bytes))
}

// NewFromInt returns a Size for the given number of bytes
func NewFromInt(bytes *big.Int) Size {
	return Size{value: bytes}
}

// Bytes returns a Size for the given number of bytes (B)
func Bytes(bytes int64) Size {
	return New(bytes)
}

// Kilobytes returns a Size for the given number of kilobytes (kB)
func Kilobytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, kilobytes))
}

// Megabytes returns a Size for the given number of megabytes (MB)
func Megabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, megabytes))
}

// Gigabytes returns a Size for the given number of gigabytes (GB)
func Gigabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, gigabytes))
}

// Terabytes returns a Size for the given number of terabytes (TB)
func Terabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, terabytes))
}

// Petabytes returns a Size for the given number of petabytes (PB)
func Petabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, petabytes))
}

// Exabytes returns a Size for the given number of exabytes (EB)
func Exabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, exabytes))
}

// Zettabytes returns a Size for the given number of zettabytes (ZB)
func Zettabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, zettabytes))
}

// Yottabytes returns a Size for the given number of yottabytes (YB)
func Yottabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, yottabytes))
}

// Kibibytes returns a Size for the given number of kibibytes (KiB)
func Kibibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, kibibytes))
}

// Mebibytes returns a Size for the given number of mebibytes (MiB)
func Mebibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, mebibytes))
}

// Gibibytes returns a Size for the given number of gibibytes (GiB)
func Gibibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, gibibytes))
}

// Tebibytes returns a Size for the given number of tebibytes (TiB)
func Tebibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, tebibytes))
}

// Pebibytes returns a Size for the given number of pebibytes (PiB)
func Pebibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, pebibytes))
}

// Exbibytes returns a Size for the given number of exbibytes (EiB)
func Exbibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, exbibytes))
}

// Zebibytes returns a Size for the given number of zebibytes (ZiB)
func Zebibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, zebibytes))
}

// Yobibytes returns a Size for the given number of yobibytes (YiB)
func Yobibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, yobibytes))
}

// Bytes returns the number of bytes in s
func (s Size) Bytes() int64 {
	return s.value.Int64()
}

func scaleBigInt(scalar float64, n *big.Int) *big.Int {
	nFloat := new(big.Float).SetInt(n)
	scalarFloat := big.NewFloat(scalar)
	mult := new(big.Float).Mul(nFloat, scalarFloat)
	i, _ := mult.Int(nil)
	return i
}

func divideBigInt(numerator, denominator *big.Int) float64 {
	f, _ := new(big.Rat).SetFrac(numerator, denominator).Float64()
	return f
}

// Kilobytes returns the number of kilobytes in s
func (s Size) Kilobytes() float64 {
	return divideBigInt(s.value, kilobytes)
}

// Megabytes returns the number of megabytes in s
func (s Size) Megabytes() float64 {
	return divideBigInt(s.value, megabytes)
}

// Gigabytes returns the number of gigabytes in s
func (s Size) Gigabytes() float64 {
	return divideBigInt(s.value, gigabytes)
}

// Terabytes returns the number of terabytes in s
func (s Size) Terabytes() float64 {
	return divideBigInt(s.value, terabytes)
}

// Petabytes returns the number of petabytes in s
func (s Size) Petabytes() float64 {
	return divideBigInt(s.value, petabytes)
}

// Exabytes returns the number of exabytes in s
func (s Size) Exabytes() float64 {
	return divideBigInt(s.value, exabytes)
}

// Zettabytes returns the number of zettabytes in s
func (s Size) Zettabytes() float64 {
	return divideBigInt(s.value, zettabytes)
}

// Yottabytes returns the number of yottabytes in s
func (s Size) Yottabytes() float64 {
	return divideBigInt(s.value, yottabytes)
}

// Kibibytes returns the number of kibibytes in s
func (s Size) Kibibytes() float64 {
	return divideBigInt(s.value, kibibytes)
}

// Mebibytes returns the number of mebibytes in s
func (s Size) Mebibytes() float64 {
	return divideBigInt(s.value, mebibytes)
}

// Gibibytes returns the number of gibibytes in s
func (s Size) Gibibytes() float64 {
	return divideBigInt(s.value, gibibytes)
}

// Tebibytes returns the number of tebibytes in s
func (s Size) Tebibytes() float64 {
	return divideBigInt(s.value, tebibytes)
}

// Pebibytes returns the number of pebibytes in s
func (s Size) Pebibytes() float64 {
	return divideBigInt(s.value, pebibytes)
}

// Exbibytes returns the number of exbibytes in s
func (s Size) Exbibytes() float64 {
	return divideBigInt(s.value, exbibytes)
}

// Zebibytes returns the number of zebibytes in s
func (s Size) Zebibytes() float64 {
	return divideBigInt(s.value, zebibytes)
}

// Yobibytes returns the number of yobibytes in s
func (s Size) Yobibytes() float64 {
	return divideBigInt(s.value, yobibytes)
}

// FormatSI formats s into a value and SI unit for the next unit with a smaller magnitude
func (s Size) FormatSI() (value float64, unit string) {
	switch {
	case s.value.CmpAbs(yottabytes) != -1:
		return s.Yottabytes(), "YB"
	case s.value.CmpAbs(zettabytes) != -1:
		return s.Zettabytes(), "ZB"
	case s.value.CmpAbs(exabytes) != -1:
		return s.Exabytes(), "EB"
	case s.value.CmpAbs(petabytes) != -1:
		return s.Petabytes(), "PB"
	case s.value.CmpAbs(terabytes) != -1:
		return s.Terabytes(), "TB"
	case s.value.CmpAbs(gigabytes) != -1:
		return s.Gigabytes(), "GB"
	case s.value.CmpAbs(megabytes) != -1:
		return s.Megabytes(), "MB"
	case s.value.CmpAbs(kilobytes) != -1:
		return s.Kilobytes(), "kB"
	default:
		return float64(s.Bytes()), "B"
	}
}

// FormatIEC formats s into a value and IEC unit for the next unit with a smaller magnitude
func (s Size) FormatIEC() (value float64, unit string) {
	switch {
	case s.value.CmpAbs(yobibytes) != -1:
		return s.Yobibytes(), "YiB"
	case s.value.CmpAbs(zebibytes) != -1:
		return s.Zebibytes(), "ZiB"
	case s.value.CmpAbs(exbibytes) != -1:
		return s.Exbibytes(), "EiB"
	case s.value.CmpAbs(pebibytes) != -1:
		return s.Pebibytes(), "PiB"
	case s.value.CmpAbs(tebibytes) != -1:
		return s.Tebibytes(), "TiB"
	case s.value.CmpAbs(gibibytes) != -1:
		return s.Gibibytes(), "GiB"
	case s.value.CmpAbs(mebibytes) != -1:
		return s.Mebibytes(), "MiB"
	case s.value.CmpAbs(kibibytes) != -1:
		return s.Kibibytes(), "KiB"
	default:
		return float64(s.Bytes()), "B"
	}
}

// String returns an SI formatted representation of s
func (s Size) String() string {
	value, unit := s.FormatSI()
	return fmt.Sprintf("%g %s", value, unit)
}
