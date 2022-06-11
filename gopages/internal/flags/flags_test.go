package flags

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()
	args, output, err := Parse("-help")
	assert.Equal(t, Args{
		OutputPath: "dist",
	}, args)
	assert.NotEmpty(t, output)
	assert.Equal(t, flag.ErrHelp, err)
}

func TestArgsLinker(t *testing.T) {
	t.Parallel()
	const someBaseURL = "/some/base"

	t.Run("default behavior - no template provided", func(t *testing.T) {
		t.Parallel()
		args := Args{
			BaseURL:            someBaseURL,
			SourceLinkTemplate: "",
		}
		linker, err := args.Linker("not used")
		require.NoError(t, err)
		assert.Equal(t, newGoPagesLinker(someBaseURL), linker)
	})

	t.Run("link template provided", func(t *testing.T) {
		t.Parallel()
		const someModulePackage = "github.com/org/repo"
		args := Args{
			BaseURL:            someBaseURL,
			SourceLinkTemplate: "{{.Path}}#L{{.Line}}",
		}
		linker, err := args.Linker(someModulePackage)
		require.NoError(t, err)
		expectLinker, err := newTemplateLinker(someModulePackage, args.SourceLinkTemplate)
		require.NoError(t, err)
		assert.Equal(t, expectLinker, linker)
	})
}
