# datasize <a href="https://johnstarich.com/go/datasize"><img src="https://img.shields.io/badge/gopages-reference-%235272B4" /></a>

Parse, format, and convert to differents units in bytes.

```go
import "github.com/johnstarich/go/datasize"

gigs := datasize.Gigabytes(1000)
fmt.Println("1000 Gigabytes equals", gigs.String())
// 1000 Gigabytes equals 1 TB
```
