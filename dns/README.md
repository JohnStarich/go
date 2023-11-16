# dns <a href="https://johnstarich.com/go/dns"><img src="https://img.shields.io/badge/gopages-reference-%235272B4" /></a>

**NOTE:** This module is obsolete as of Go 1.20. Go now uses the [native macOS DNS resolver][golang-issue]!

Implements a working macOS DNS resolver (really just a `Dial`er) for projects that must cross-compile from Linux systems or just don't want CGO.

In my experience, it is common to disable CGO for macOS CI builds. However, there's been a [few issues with that][golang-issue]. This library adds a drop-in replacement for Go's `net.DefaultResolver` to fill the gap.

```go
import _ "github.com/johnstarich/go/dns/init"
```

This resolver reads the system's full DNS configuration and attempts to find the a successful nameserver. First, the dialer reaches out to the default nameserver. If the response isn't fast enough, more nameservers are attempted simultaneously.

On non-macOS builds a normal resolver is used, so this is safe to use for multi-platform builds.

Thoughts or questions? Please [open an issue](https://github.com/JohnStarich/go/issues/new) to discuss.

[golang-issue]: https://github.com/golang/go/issues/12524

## Debugging

Sometimes correct DNS results depends on nameservers being tried in a very specific order.
If you see the wrong nameserver's results, try tuning the `Config`'s timing settings.

For example, if the first nameserver is likely correct but is slow to respond, then increase the `InitialNameserverDelay` to compensate.
