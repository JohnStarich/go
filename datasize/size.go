package datasize

import (
	"fmt"
	"math/big"
)

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

type Size struct {
	value *big.Int
}

func New(bytes int64) Size {
	return NewFromInt(big.NewInt(bytes))
}

func NewFromInt(bytes *big.Int) Size {
	return Size{value: bytes}
}

func Bytes(bytes int64) Size {
	return New(bytes)
}

func Kilobytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, kilobytes))
}

func Megabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, megabytes))
}

func Gigabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, gigabytes))
}

func Terabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, terabytes))
}

func Petabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, petabytes))
}

func Exabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, exabytes))
}

func Zettabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, zettabytes))
}

func Yottabytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, yottabytes))
}

func Kibibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, kibibytes))
}

func Mebibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, mebibytes))
}

func Gibibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, gibibytes))
}

func Tebibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, tebibytes))
}

func Pebibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, pebibytes))
}

func Exbibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, exbibytes))
}

func Zebibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, zebibytes))
}

func Yobibytes(b float64) Size {
	return NewFromInt(scaleBigInt(b, yobibytes))
}

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

func (s Size) Kilobytes() float64 {
	return divideBigInt(s.value, kilobytes)
}

func (s Size) Megabytes() float64 {
	return divideBigInt(s.value, megabytes)
}

func (s Size) Gigabytes() float64 {
	return divideBigInt(s.value, gigabytes)
}

func (s Size) Terabytes() float64 {
	return divideBigInt(s.value, terabytes)
}

func (s Size) Petabytes() float64 {
	return divideBigInt(s.value, petabytes)
}

func (s Size) Exabytes() float64 {
	return divideBigInt(s.value, exabytes)
}

func (s Size) Zettabytes() float64 {
	return divideBigInt(s.value, zettabytes)
}

func (s Size) Yottabytes() float64 {
	return divideBigInt(s.value, yottabytes)
}

func (s Size) Kibibytes() float64 {
	return divideBigInt(s.value, kibibytes)
}

func (s Size) Mebibytes() float64 {
	return divideBigInt(s.value, mebibytes)
}

func (s Size) Gibibytes() float64 {
	return divideBigInt(s.value, gibibytes)
}

func (s Size) Tebibytes() float64 {
	return divideBigInt(s.value, tebibytes)
}

func (s Size) Pebibytes() float64 {
	return divideBigInt(s.value, pebibytes)
}

func (s Size) Exbibytes() float64 {
	return divideBigInt(s.value, exbibytes)
}

func (s Size) Zebibytes() float64 {
	return divideBigInt(s.value, zebibytes)
}

func (s Size) Yobibytes() float64 {
	return divideBigInt(s.value, yobibytes)
}

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

func (s Size) String() string {
	value, unit := s.FormatSI()
	return fmt.Sprintf("%g %s", value, unit)
}
