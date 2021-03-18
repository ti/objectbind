# objectbind

Object->files mapping,  dynamic bind files, directory, etcd or other file-like backend to golang object

### Simple Interface to Use

* `Bind` bind the object to file or etcd
* `BindField` receive field changes when the file or etcd is change
* `Save` save the local object to remote

### Distributed files supported

For example:

```go
defaultConfig := struct {
    Name map[string]interface{}
    Data []map[string]interface{} `bind:"data/conf/test"`
}{}

objectbind.Bind(ctx, &defaultConfig, "conf/test.yaml")
```

`conf/test.yaml` will map to `struct {Name map[string]interface{}}`

`data/conf/test.yaml` will map to `struct {Data []map[string]interface{}`

### example

```go
package main

import (
	"context"
	"fmt"
	"github.com/ti/objectbind"
)

type Config struct {
	Name map[string]interface{}
	Data []map[string]interface{} `bind:"data/conf/test"`
	Data2 map[string]interface{}  `bind:"etc/data/conf2/"`
}


func main() {
	defaultConfig := Config{
		Name: map[string]interface{}{
			"Jhon": "smith",
		},
	}
	ctx := context.Background()

	// 1. Bind bind data to the file
	binder, _ := objectbind.Bind(ctx, &defaultConfig, "conf/test.yaml")

	// 2. BindField receive data change when Name filed is change
	binder.BindField("Name", func(value, _ interface{}) {
		fmt.Println("GET Data", value.(map[string]interface{}))
	})

	// change the data
	defaultConfig.Name["hello"] = "world"

	// 3. Save save the config changes  to remote
	_ = binder.Save(ctx)
}
```
