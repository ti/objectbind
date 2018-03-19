package objectbind

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
)

//mapData the key value of a interface
type mapData struct {
	// the path of the file
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type fieldKind struct {
	Field string
	Kind  reflect.Kind
}

// marshal unmarshal interface to kv array
func marshal(key string, target interface{}, tagName string) (kvs []*mapData, err error) {
	src := getReflectValue(target)
	mainKind := src.Kind()
	ifMainIsDir := strings.HasSuffix(key, "/")
	if mainKind == reflect.Struct {
		rootKey := key
		if !strings.HasSuffix(rootKey, "/") {
			i := strings.LastIndex(rootKey, "/")
			if i > 0 {
				rootKey = rootKey[0 : i+1]
			}
		}
		size := src.NumField()
		var mainKvs []*mapData
		for i := 0; i < size; i++ {
			f := src.Type().Field(i)
			jsonFiledKey := getFiledTag("json", &f)
			fKey := getFiledTag(tagName, &f)
			data := src.Field(i).Interface()
			var childKvs []*mapData
			var mainChildKvs []*mapData
			if strings.Contains(fKey, "/") {
				if !strings.HasPrefix(fKey, "/") {
					fKey = rootKey + fKey
				}
				childKvs, err = uniMarshal(fKey, &f, data)
			} else {
				if ifMainIsDir {
					childKvs, err = simpleKVMarshal(rootKey+jsonFiledKey, data)
				} else {
					mainChildKvs, err = simpleKVMarshal(jsonFiledKey, data)
				}
			}
			if err != nil {
				return nil, err
			}
			kvs = append(kvs, childKvs...)
			mainKvs = append(mainKvs, mainChildKvs...)
		}
		if len(mainKvs) > 0 {
			value := kvsToJSON(mainKvs)
			data := &mapData{
				Key:   key,
				Value: value,
			}
			kvs = append([]*mapData{
				data,
			}, kvs...)
		}
	} else {
		if !ifMainIsDir {
			kvs, err = simpleKVMarshal(key, target)
		} else {
			switch mainKind {
			case reflect.Map:
				kvs, err = mapKVMarshal(key, src)
			case reflect.Slice:
				kvs, err = sliceKVMarshal(key, src)
			default:
				kvs, err = simpleKVMarshal(key, target)
			}
		}
	}
	return
}

func getFiledTag(tagName string, f *reflect.StructField) string {
	fKey := f.Name
	if t := f.Tag.Get(tagName); t != "" {
		indexDot := strings.Index(t, ",")
		if indexDot < 0 {
			fKey = t
		} else if indexDot > 0 {
			fKey = t[:indexDot]
		} else {
			fKey = strings.Split(t, ",")[0]
		}
	}
	return fKey
}

func kvsToJSON(kvs []*mapData) string {
	ret := "{"
	kvsLen := len(kvs)
	for i, data := range kvs {
		ret += `"` + data.Key + `":` + data.Value
		if i < kvsLen-1 {
			ret += ","
		}
	}
	ret += "}"
	return ret
}

func getReflectValue(i interface{}) reflect.Value {
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		return reflect.ValueOf(reflect.Indirect(reflect.ValueOf(i)).Interface())
	}
	return reflect.ValueOf(i)
}

func getReflectInterface(data interface{}) (kind reflect.Kind, i interface{}) {
	v := reflect.ValueOf(data)
	k := v.Kind()
	if k == reflect.Ptr {
		i = reflect.Indirect(v).Interface()
	} else {
		i = data
	}
	kind = reflect.ValueOf(i).Kind()
	return
}

func mapKVMarshal(key string, src reflect.Value) (kvs []*mapData, err error) {
	keys := src.MapKeys()
	for _, k := range keys {
		value := src.MapIndex(k)
		b, err := json.Marshal(value.Interface())
		if err != nil {
			return nil, err
		}
		kvs = append(kvs, &mapData{
			Key:   key + k.String(),
			Value: string(b),
		})
	}
	return
}

func sliceKVMarshal(key string, src reflect.Value) (kvs []*mapData, err error) {
	size := src.Len()
	for i := 0; i < size; i++ {
		value := src.Index(i)
		b, err := json.Marshal(value.Interface())
		if err != nil {
			return nil, err
		}
		kvs = append(kvs, &mapData{
			Key:   key + strconv.Itoa(i),
			Value: string(b),
		})
	}
	return
}

func simpleKVMarshal(key string, target interface{}) ([]*mapData, error) {
	b, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	data := &mapData{
		Key:   key,
		Value: string(b),
	}
	return []*mapData{
		data,
	}, nil
}

func convertMainData(target interface{}, tagName string) map[string]interface{} {
	data := make(map[string]interface{})
	src := getReflectValue(target)
	mainKind := src.Kind()
	if mainKind == reflect.Struct {
		size := src.NumField()
		for i := 0; i < size; i++ {
			f := src.Type().Field(i)
			jsonFiledKey := getFiledTag("json", &f)
			if t := f.Tag.Get(tagName); t == "" {
				data[jsonFiledKey] = src.Field(i).Interface()
			}
		}
	}
	return data
}

func uniMarshal(fKey string, f *reflect.StructField, data interface{}) (childKvs []*mapData, err error) {
	fKind := f.Type.Kind()
	if !strings.HasSuffix(fKey, "/") {
		childKvs, err = simpleKVMarshal(fKey, data)
	} else {
		switch fKind {
		case reflect.Map:
			childKvs, err = mapKVMarshal(fKey, getReflectValue(data))
		case reflect.Slice:
			childKvs, err = sliceKVMarshal(fKey, getReflectValue(data))
		default:
			childKvs, err = simpleKVMarshal(fKey, data)
		}
	}
	return
}
