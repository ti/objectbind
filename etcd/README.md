# how to import ETCD Plugin

1. in your main project folder

get lasted go.etcd.io/etcd/client/v3 package

```bash
go get go.etcd.io/etcd/client/v3@v3.5.0
```

2.  Edit your main.go

```go
package main

import (
	_ "github.com/ti/objectbind/etcd"
)
```

the etcd plugin will be auto registed


