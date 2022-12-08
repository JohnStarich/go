# plist <a href="https://johnstarich.com/go/plist"><img src="https://img.shields.io/badge/gopages-reference-%235272B4" /></a>

Plist parses macOS `*.plist` input data and formats them into JSON.

NOTE: Currently only supports XML based plists, not the binary format.

## Getting started

Try out plist with a small program like this:
```go
package main

import "github.com/johnstarich/go/plist"

func main() {
    myPlistContents := `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Greeting</key>
    <string>Hello world!</string>
</dict>
`
    reader := strings.NewReader(myPlistContents)
    jsonBytes, _ := plist.ToJSON(reader)

    var myPList MyPList
    _ = json.Unmarshal(jsonBytes, &myPList)
    fmt.Println(myPList)
    // Output: {Hello world!}
}

type MyPList struct {
    Greeting string
}
```

Thoughts or questions? Please [open an issue](https://github.com/JohnStarich/go/issues/new) to discuss.
