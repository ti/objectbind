# objectbind

Object-files mapping,  dynamic bind files, directory, etcd or other file-like backend to golang object

```go
type Config struct {
	Name map[string]interface{}
	Data []map[string]interface{} `bind:"data/conf/test"`
	Data2 map[string]interface{}  `bind:"etc/data/conf2/"`
}
var cfg Config{}

// bind data to 
binder, _ := objectbind.Bind(ctx, &cfg, "conf/test.yaml")

// receive data change when Name filed is change
binder.BindField("Name", func(value, _ interface{}) {
    fmt.Println("GET Data", value.(map[string]interface{}))
})

// save Save the config chanages  to remote
binder.Save(ctx))

```
