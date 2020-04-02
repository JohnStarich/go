package datasize

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert.Equal(t, Size{value: big.NewInt(2)}, New(2))
}

func TestBytes(t *testing.T) {
	assert.Equal(t, int64(2), Bytes(2).Bytes())
}

func TestKilobytes(t *testing.T) {
	assert.Equal(t, New(2e3), Kilobytes(2))
}

func TestMegabytes(t *testing.T) {
	assert.Equal(t, New(2e6), Megabytes(2))
}

func TestGigabytes(t *testing.T) {
	assert.Equal(t, New(2e9), Gigabytes(2))
}

func TestTerabytes(t *testing.T) {
	assert.Equal(t, New(2e12), Terabytes(2))
}

func TestPetabytes(t *testing.T) {
	assert.Equal(t, New(2e15), Petabytes(2))
}

func TestExabytes(t *testing.T) {
	assert.Equal(t, New(2e18), Exabytes(2))
}

func TestZettabytes(t *testing.T) {
	zetta := new(big.Int).Mul(big.NewInt(2e18), big.NewInt(1e3))
	assert.Equal(t, NewFromInt(zetta), Zettabytes(2))
}

func TestYottabytes(t *testing.T) {
	yotta := new(big.Int).Mul(big.NewInt(2e18), big.NewInt(1e6))
	assert.Equal(t, NewFromInt(yotta), Yottabytes(2))
}

func TestKibibytes(t *testing.T) {
	assert.Equal(t, New(2<<10), Kibibytes(2))
}

func TestMebibytes(t *testing.T) {
	assert.Equal(t, New(2<<20), Mebibytes(2))
}

func TestGibibytes(t *testing.T) {
	assert.Equal(t, New(2<<30), Gibibytes(2))
}

func TestTebibytes(t *testing.T) {
	assert.Equal(t, New(2<<40), Tebibytes(2))
}

func TestPebibytes(t *testing.T) {
	assert.Equal(t, New(2<<50), Pebibytes(2))
}

func TestExbibytes(t *testing.T) {
	assert.Equal(t, New(2<<60), Exbibytes(2))
}

func TestZebibytes(t *testing.T) {
	zebi := new(big.Int).Mul(big.NewInt(2<<60), big.NewInt(1<<10))
	assert.Equal(t, NewFromInt(zebi), Zebibytes(2))
}

func TestYobibytes(t *testing.T) {
	yobi := new(big.Int).Mul(big.NewInt(2<<60), big.NewInt(1<<20))
	assert.Equal(t, NewFromInt(yobi), Yobibytes(2))
}

func TestFormatSI(t *testing.T) {
	for _, tc := range []struct {
		input         float64
		expectedValue float64
		expectedUnit  string
	}{
		{0, 0, "B"},
		{1, 1, "B"},
		{2e3, 2, "kB"},
		{2345, 2.345, "kB"},
		{2e6, 2, "MB"},
		{7.893e6, 7.893, "MB"},
		{2e9, 2, "GB"},
		{2e12, 2, "TB"},
		{2e15, 2, "PB"},
		{2e18, 2, "EB"},
		{2e21, 2, "ZB"},
		{2e24, 2, "YB"},
	} {
		t.Run(fmt.Sprintf("%g", tc.input), func(t *testing.T) {
			i, _ := big.NewFloat(tc.input).Int(nil)
			num := NewFromInt(i)
			value, unit := num.FormatSI()
			assert.Equal(t, tc.expectedValue, value)
			assert.Equal(t, tc.expectedUnit, unit)

			assert.Equal(t, fmt.Sprintf("%g %s", tc.expectedValue, tc.expectedUnit), num.String())
		})
	}
}

func TestFormatIEC(t *testing.T) {
	for _, tc := range []struct {
		input         float64
		expectedValue float64
		expectedUnit  string
	}{
		{0, 0, "B"},
		{1, 1, "B"},
		{2 << 10, 2, "KiB"},
		{2345, 2.290, "KiB"},
		{2 << 20, 2, "MiB"},
		{7893 << 20, 7.708, "GiB"},
		{2 << 30, 2, "GiB"},
		{2 << 40, 2, "TiB"},
		{2 << 50, 2, "PiB"},
		{2 << 60, 2, "EiB"},
		{2 << 70, 2, "ZiB"},
		{2 << 80, 2, "YiB"},
	} {
		t.Run(fmt.Sprintf("%g", tc.input), func(t *testing.T) {
			i, _ := big.NewFloat(tc.input).Int(nil)
			num := NewFromInt(i)
			value, unit := num.FormatIEC()
			assert.InDelta(t, tc.expectedValue, value, 0.0001)
			assert.Equal(t, tc.expectedUnit, unit)
		})
	}
}
