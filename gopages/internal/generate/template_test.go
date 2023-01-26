package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/godoc"
)

func TestNodeHTML(t *testing.T) {
	t.Parallel()
	htmlFunc := func(*godoc.PageInfo, interface{}, bool) string {
		return `
<html>
<body>

This module's package:
<a href="/pkg/github.com/org/repo/package/foo.go">foo.go</a>
<a href="/pkg/github.com/org/repo/package/sub-package/bar.go">bar.go</a>
<a href="/pkg/github.com/org/repo/package">package</a>

Standard library package:
<a href="/pkg/os">os</a>
<a href="/pkg/os/file.go">file.go</a>

Unrecognized package:
<a href="/pkg/github.com/not/my/package/foo.go">foo.go</a>
<a href="/pkg/github.com/not/my/package/sub-package/bar.go">bar.go</a>
<a href="/pkg/github.com/not/my/package">package</a>

</body>
</html>
`
	}
	newHTML := nodeHTML(htmlFunc, "/base", "github.com/org/repo/package")(nil, nil, false)
	assert.Equal(t, `
<html>
<body>

This module's package:
<a href="/base/pkg/github.com/org/repo/package/foo.go">foo.go</a>
<a href="/base/pkg/github.com/org/repo/package/sub-package/bar.go">bar.go</a>
<a href="/base/pkg/github.com/org/repo/package">package</a>

Standard library package:
<a href="https://pkg.go.dev/os">os</a>
<a href="https://pkg.go.dev/os/file.go">file.go</a>

Unrecognized package:
<a href="https://pkg.go.dev/github.com/not/my/package/foo.go">foo.go</a>
<a href="https://pkg.go.dev/github.com/not/my/package/sub-package/bar.go">bar.go</a>
<a href="https://pkg.go.dev/github.com/not/my/package">package</a>

</body>
</html>
`, newHTML)
}

func TestCommentHTML(t *testing.T) {
	t.Parallel()
	htmlFunc := func(*godoc.PageInfo, string) string {
		return `
<html>
<body>

This module's package:
<a href="/github.com/org/repo/package/foo.go">foo.go</a>
<a href="/github.com/org/repo/package/sub-package/bar.go">bar.go</a>
<a href="/github.com/org/repo/package">package</a>

Standard library package:
<a href="/os">os</a>
<a href="/os/file.go">file.go</a>

Unrecognized package:
<a href="/github.com/not/my/package/foo.go">foo.go</a>
<a href="/github.com/not/my/package/sub-package/bar.go">bar.go</a>
<a href="/github.com/not/my/package">package</a>

</body>
</html>
`
	}
	newHTML := commentToHTML(htmlFunc, "/base", "github.com/org/repo/package")(nil, "")
	assert.Equal(t, `
<html>
<body>

This module's package:
<a href="/base/pkg/github.com/org/repo/package/foo.go">foo.go</a>
<a href="/base/pkg/github.com/org/repo/package/sub-package/bar.go">bar.go</a>
<a href="/base/pkg/github.com/org/repo/package">package</a>

Standard library package:
<a href="https://pkg.go.dev/os">os</a>
<a href="https://pkg.go.dev/os/file.go">file.go</a>

Unrecognized package:
<a href="https://pkg.go.dev/github.com/not/my/package/foo.go">foo.go</a>
<a href="https://pkg.go.dev/github.com/not/my/package/sub-package/bar.go">bar.go</a>
<a href="https://pkg.go.dev/github.com/not/my/package">package</a>

</body>
</html>
`, newHTML)
}
