# regext <a href="https://johnstarich.com/go/regext"><img src="https://img.shields.io/badge/gopages-reference-%235272B4" /></a>

Extended regex, so you can *almost* read regular expressions. 😉

Compiles regular expressions and ignores whitespace and in-line comments, so it's easier to understand the expression.

```go
myExpression := regext.MustCompile(`
    \w+           # Ignore first name
    \s+ (\w+)?    # Capture last name, ignore leading spaces
`)
matches := myExpression.FindStringSubmatch("John Doe")
fmt.Println(matches[1])
// Doe
```
