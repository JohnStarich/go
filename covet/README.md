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


## Integrate with GitHub

Covet includes an automatic GitHub Actions integration and opt-in GitHub comment summary reports.

To enable comment summaries, you need to set a few additional flags.

Here's an example using GitHub Actions:
```yaml
  test:
    runs-on: ubuntu-latest
    permissions:
      pull-requests: write
    steps:
    - uses: actions/checkout@v3
    - name: checkout
      run: |
        commits=${{ github.event.pull_request.commits }}
        if [[ -n "$commits" ]]; then
          # Prepare enough depth for diffs with master
          git fetch --depth="$(( commits + 1 ))"
        fi
    - uses: actions/setup-go@v3
      with:
        go-version: 1.16.x
    - name: Test
      run: |
	git diff origin/master | \
	    covet \
		-diff-file - \
		-cover-go ./cover.out \
		-show-diff-coverage \
		-gh-token "$GITHUB_TOKEN" \
		-gh-issue "github.com/${GITHUB_REPOSITORY}/pull/${ISSUE_NUMBER}" 
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

And here's an example using environment variables from Travis CI:
```bash
git diff origin/master | \
    covet \
        -diff-file - \
        -cover-go ./cover.out \
        -show-diff-coverage \
        -gh-token "$GITHUB_TOKEN" \
        -gh-issue "github.com/${TRAVIS_PULL_REQUEST_SLUG}/pull/${TRAVIS_PULL_REQUEST}"
```
