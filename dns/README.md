# dns

Implements a working macOS DNS resolver (really just a `Dial`er) for projects that must cross-compile from Linux systems or just don't want CGO.

In my experience, it is common to disable CGO for macOS CI builds. However, there's been a [few issues with that][golang-issue]. This library adds a drop-in replacement for Go's `net.DefaultResolver` to fill the gap.

```go
import _ "github.com/johnstarich/go/dns/init"
```

This resolver reads the system's full DNS configuration and attempts to find the a successful nameserver. First, the dialer reaches out to the default nameserver. If the response isn't fast enough, more nameservers are attempted simultaneously.

On non-macOS builds a normal resolver is used, so this is safe to use for multi-platform builds.

Thoughts or questions? Please [open an issue](https://github.com/JohnStarich/go/issues/new) to discuss.

[golang-issue]: https://github.com/golang/go/issues/12524
