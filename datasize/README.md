# datasize

Parse, format, and convert to differents units in bytes.

```go
import "github.com/johnstarich/go/datasize"

gigs := datasize.Gigabytes(1000)
fmt.Println("1000 Gigabytes equals", gigs.String())
// 1000 Gigabytes equals 1 TB
```
