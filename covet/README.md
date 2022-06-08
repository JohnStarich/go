# covet <a href="https://johnstarich.com/go/covet"><img src="https://img.shields.io/badge/gopages-reference-%235272B4" /></a>

Cover it!

`covet` reads version control diffs and Go coverage files to generate useful stats about their intersection.
Currently displays coverage percentage of the diff and highlights any new, uncovered lines.

```bash
go install github.com/johnstarich/go/covet/cmd/covet@latest
covet -help

git diff HEAD~10...HEAD > my.diff
go test -coverprofile=cover.out ./...

covet -diff-file my.diff -cover-go cover.out
#  ... includes diff and coverage intersection ...
# Total diff coverage:  84.1%
#
# Diff coverage is below target. Add tests for these files:
#
# ┌─────────┬──────────────┬────────────────────────────────────────────┐
# │ LINES   │ COVERAGE     │ FILE                                       │
# ├─────────┼──────────────┼────────────────────────────────────────────┤
# │ 239/279 │  85.7% ████▎ │ diffcover/cmd/diffcover/run.go             │
# │  16/42  │  38.1% █▉    │ diffcover/cmd/diffcover/coverage_status.go │
# └─────────┴──────────────┴────────────────────────────────────────────┘
```

Still experimental: Future releases may contain breaking changes.

Thoughts or questions? Please [open an issue](https://github.com/JohnStarich/go/issues/new) to discuss.
