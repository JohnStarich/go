# goop <a href="https://johnstarich.com/go/goop"><img src="https://img.shields.io/badge/gopages-reference-%235272B4" /></a>

Goop makes it easy to write and run Go scripts. Automatically builds installed Go commands when you run them.

Features:
* Support for both local and remote modules.
* Automatic rebuilds of local modules.
* Shareable command bin for easy setup on multiple machines.

## Getting started

1. Install `goop` - `go install github.com/johnstarich/go/goop/cmd/goop@latest`
2. Read the built-in documentation - `goop --help`
3. Install a module. (Check for warnings in output.) - `goop install -p github.com/johnstarich/go/covet/cmd/covet@latest`
4. Run the module by name to execute it - `covet --help`

Thoughts or questions? Please [open an issue](https://github.com/JohnStarich/go/issues/new) to discuss.
