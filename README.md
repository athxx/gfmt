# `gfmt` a go format tool
`gfmt` rely on `gofmt` and better than `goimports` and `gofmt`, it'll
rearrange all packages import and move into different groups. If your teams
using difference IDE and always imports not arrange, maybe `gfmt` will be
your first choice. it's base on execute command `gofmt -s -w` and then
to format your imports.

## Install
> go install github.com/athxx/gfmt
> 
## How to use

```shell
# gfmt [file or path]
gfmt .
```

# What different with "goimports" and "gofmt"

`goimport` and  `gofmt` will only format code but not align imports and groups packages

but `gfmt` will format and group up package

```go
# goimport and gofmt would not align
package main

import (
	"fmt"
	//"strconv"
	"github.com/gorilla/mux"
	"strings"
)

func main() {
	var _ = mux.NewRouter()
	fmt.Println(strings.ToUpper("Hello, world!"))
}
```

but `gfmt`

```go
package main

import (
	"fmt"
	"strings"

	"github.com/gorilla/mux"

	// "strconv"     <- all annotation line will group up 
)

func main() {
	var _ = mux.NewRouter()
	fmt.Println(strings.ToUpper("Hello, world!"))
}
```

