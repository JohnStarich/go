package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserBinDir(t *testing.T) {
	t.Parallel()
	const staticBin = "bin"
	t.Run("static bin", func(t *testing.T) {
		t.Parallel()
		app := App{
			getEnv: func(s string) string {
				return ""
			},
			staticBinDir: staticBin,
		}
		dir, err := app.userBinDir()
		assert.NoError(t, err)
		assert.Equal(t, staticBin, dir)
	})

	t.Run("environment bin", func(t *testing.T) {
		t.Parallel()
		const envBin = "homes/joe/bin"
		app := App{
			getEnv: func(s string) string {
				if s == "GOOP_BIN" {
					return envBin
				}
				return ""
			},
			staticBinDir: staticBin,
		}
		dir, err := app.userBinDir()
		assert.NoError(t, err)
		assert.Equal(t, envBin, dir)
	})
}
