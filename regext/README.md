# regext

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
