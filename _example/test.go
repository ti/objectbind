package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ti/objectbind"
)

type Config struct {
	Version          string
	Data             map[string]interface{}
	DataMap          map[string]interface{}             `bind:"data/conf/map"`
	DataMapRoot      map[string]interface{}             `bind:"data/conf/maproot/"`
	DataMapRootPtr   map[string]*map[string]interface{} `bind:"data/conf/maprootptr/"`
	DataSlice        []map[string]interface{}           `bind:"data/conf/slice"`
	DataSliceRoot    []map[string]interface{}           `bind:"data/conf/sliceroot/"`
	DataSliceRootPtr []*map[string]interface{}          `bind:"data/conf/slicerootptr/"`
}

func main() {
	var cfg = Config{
		Version: "v1.1",
		Data: map[string]interface{}{
			"d1": "haha2",
			"d2": "haha2",
			"d3": "haha2",
		},
		DataMap: map[string]interface{}{
			"Name2": "haha2",
		},
		DataMapRoot: map[string]interface{}{
			"m1": "haha2",
			"m2": "haha2",
			"m3": "haha2",
		},
		DataSlice: []map[string]interface{}{{
			"s1":   "haha2",
			"s1.1": "haha2",
		}},
		DataSliceRoot: []map[string]interface{}{
			{
				"sr1":   "haha2",
				"sr1.1": "haha2",
			},
			{
				"sr2":   "haha2",
				"sr2.1": "haha2",
			},
			{
				"sr3":   "haha2",
				"sr3.1": "haha2",
			},
			{
				"sr4":   "haha2",
				"sr4.1": "haha2",
			},
			{
				"sr5":   "haha2",
				"sr5.1": "haha2",
			},
			{
				"sr6":   "haha2",
				"sr6.1": "haha2",
			},
			{
				"s7":   "haha2",
				"s7.1": "haha2",
			},
			{
				"sr8":   "haha2",
				"sr8.1": "haha2",
			},
			{
				"sr9":   "haha2",
				"sr9.1": "haha2",
			},
			{
				"sr10":   "haha2",
				"sr10.1": "haha2",
			},
			{
				"sr11":   "haha2",
				"se11.1": "haha2",
			},
			{
				"sr12":   "haha2",
				"sr12.1": "haha2",
			},
		},
	}

	ctx := context.Background()
	binder, err := objectbind.Bind(ctx, &cfg, "test/conf/yes.yaml")
	if err != nil {
		panic(err)
	}
	binder.BindField("Version", func(value, _ interface{}) {
		fmt.Println("GET Version", value.(string))
	})

	binder.BindField("Data", func(value, _ interface{}) {
		fmt.Println("GET Data", value.(map[string]interface{}))
	})
	binder.BindField("DataMap", func(value, _ interface{}) {
		fmt.Println("Get DataMap", value.(map[string]interface{}))
	})
	binder.BindField("DataMapRoot", func(value, _ interface{}) {
		fmt.Println("GET DataMapRoot", value.(map[string]interface{}))
	})

	binder.BindField("DataSliceRoot[3]", func(value, _ interface{}) {
		fmt.Println("GET DataMapRoot[3]", value.(map[string]interface{}))
	})

	binder.BindField("DataSliceRoot", func(value, _ interface{}) {
		fmt.Println("GET DataSliceRoot", value.([]map[string]interface{}))
	})

	binder.BindField("DataSlice[0]", func(value, _ interface{}) {
		fmt.Println("GET DataSlice[0]", value.(map[string]interface{}))
	})

	go func() {
		time.Sleep(3 * time.Second)
		fmt.Println("Save Data")
		cfg.DataSliceRoot = append(cfg.DataSliceRoot, map[string]interface{}{
			"test": "ok - " + time.Now().Format(time.RFC3339),
		})

		cfg.DataMapRootPtr = make(map[string]*map[string]interface{})
		cfg.DataMapRootPtr["test"] = &map[string]interface{}{
			"test": "ok - " + time.Now().Format(time.RFC3339),
		}

		err := binder.Save(ctx)
		fmt.Println(err)
	}()

	//err = binder.Save(context.Background())
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Hour)
}
