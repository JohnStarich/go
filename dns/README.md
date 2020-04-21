# dns

Implements a macOS DNS resolver (really just the `Dial`er) for projects that must cross-compile from non-macOS systems or disable CGO.

In my experience, it is common to disable CGO for macOS CI builds. However, there's been a few issues with that. This library adds a drop-in replacement for Go's `net.DefaultResolver` to fill the gap.

This resolver reads the system's DNS configuration to pick up and call out to all known DNS nameservers.
