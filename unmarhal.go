package objectbind

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

//unmarshal unmarshal interface to kv array
func unmarshal(key string, kvs []*mapData, target interface{}, tagName string) (err error) {
	if len(kvs) == 0 {
		return errors.New("kvs size is 0")
	}
	src := getReflectValue(target)
	mainKind := src.Kind()
	ifMainIsDir := strings.HasSuffix(key, "/")
	if mainKind == reflect.Struct {
		var js string
		js, err = unmarshalKVStructToJson(key, target, kvs, tagName)
		if err == nil {
			err = json.Unmarshal([]byte(js), target)
		}
	} else {
		if !ifMainIsDir {
			err = json.Unmarshal([]byte(kvs[0].Value), target)
		} else {
			switch mainKind {
			case reflect.Map:
				err = mapKVUnmarshal(key, kvs, target)
			case reflect.Slice:
				err = sliceKVUnmarshal(key, kvs, target)
			default:
				err = json.Unmarshal([]byte(kvs[0].Value), target)
			}
		}
	}
	return
}

func unmarshalKVStructToJson(path string, target interface{}, kvs []*mapData, tagName string) (js string, err error) {
	rootKey := path
	if !strings.HasSuffix(rootKey, "/") {
		i := strings.LastIndex(rootKey, "/")
		if i > 0 {
			rootKey = rootKey[0 : i+1]
		}
	}
	var keysKind map[string]fieldKind
	keysKind, err = getKeysKind(rootKey, target, tagName)
	if err != nil {
		return
	}
	type KindKVs struct {
		KVS   []*mapData
		Kind  reflect.Kind
		Field string
	}
	var childKvs = make(map[string]*KindKVs)
	var parts []string
	for _, data := range kvs {
		fKey := data.Key
		fValue := data.Value
		if len(fValue) == 0 {
			continue
		}
		if fKey == path {
			parts = append(parts, fValue[1:len(fValue)-1])
		} else if !strings.Contains(fKey, "/") {
			parts = append(parts, `"`+fKey+`":`+fValue)
		} else if strings.HasPrefix(fKey, rootKey) {
			fKey = fKey[len(rootKey):]
			index := strings.LastIndex(fKey, "/")
			if index > 0 {
				childKey := fKey[0 : index+1]
				childKind, ok := keysKind[rootKey+childKey]
				if ok {
					childKV, ok := childKvs[childKey]
					if ok {
						childKV.KVS = append(childKV.KVS, data)
					} else {
						childKvs[childKey] = &KindKVs{
							Kind:  childKind.Kind,
							Field: childKind.Field,
							KVS:   []*mapData{data},
						}
					}
				} else {
					filedFKey := rootKey + fKey
					childKind, ok := keysKind[filedFKey]
					if ok {
						parts = append(parts, `"`+childKind.Field+`":`+fValue)
					} else {
						parts = append(parts, `"`+fKey+`":`+fValue)
					}
				}
			} else {
				parts = append(parts, `"`+fKey+`":`+fValue)
			}
		} else {
			kindKey, ok := keysKind[fKey]
			if ok {
				parts = append(parts, `"`+kindKey.Field+`":`+fValue)
			} else {
				index := strings.LastIndex(fKey, "/")
				childKey := fKey[0 : index+1]
				childKind, ok := keysKind[childKey]
				if ok {
					childKV, ok := childKvs[childKey]
					if ok {
						childKV.KVS = append(childKV.KVS, data)
					} else {
						childKvs[childKey] = &KindKVs{
							Kind:  childKind.Kind,
							Field: childKind.Field,
							KVS:   []*mapData{data},
						}
					}
				}
			}
		}
	}
	for _, v := range childKvs {
		js := partsToJSON(v.Kind, v.KVS)
		parts = append(parts, `"`+v.Field+`":`+js)
	}
	js = "{" + strings.Join(parts, ",") + "}"
	return
}

func partsToJSON(kind reflect.Kind, kvs []*mapData) string {
	switch kind {
	case reflect.Map, reflect.Struct:
		ret := "{"
		maxIndex := len(kvs) - 1
		for i, data := range kvs {
			index := strings.LastIndex(data.Key, "/")
			mapKey := data.Key[index+1:]
			part := `"` + mapKey + `":` + data.Value
			if i == maxIndex {
				part += "}"
			} else {
				part += ","
			}
			ret += part
		}
		return ret
	case reflect.Slice:
		ret := "["
		maxIndex := len(kvs) - 1
		for i, data := range kvs {
			ret += data.Value
			if i == maxIndex {
				ret += "]"
			} else {
				ret += ","
			}
		}
		return ret
	default:
		return "{}"
	}
}

func mapKVUnmarshal(key string, kvs []*mapData, distValue interface{}) (err error) {
	ret := "{"
	maxIndex := len(kvs) - 1
	for i, data := range kvs {
		fKey := data.Key
		fKey = strings.TrimPrefix(fKey, key)
		if fKey != "" && !strings.Contains(fKey, "/") {
			part := `"` + fKey + `":` + data.Value
			if i == maxIndex {
				part += "}"
			} else {
				part += ","
			}
			ret += part
		}
	}
	return json.Unmarshal([]byte(ret), distValue)
}

func sliceKVUnmarshal(key string, kvs []*mapData, distValue interface{}) (err error) {
	ret := "["
	maxIndex := len(kvs) - 1
	for i, data := range kvs {
		fKey := strings.TrimPrefix(data.Key, key)
		if fKey != "" && !strings.Contains(fKey, "/") {
			ret += data.Value
			if i == maxIndex {
				ret += "]"
			} else {
				ret += ","
			}
		}
	}
	return json.Unmarshal([]byte(ret), distValue)
}

//getKeysKind get all the keys and kind for list usage
func getKeysKind(key string, target interface{}, tagName string) (keyTypeMap map[string]fieldKind, err error) {
	src := getReflectValue(target)
	mainKind := src.Kind()
	keyTypeMap = make(map[string]fieldKind)
	keyTypeMap[key] = fieldKind{
		Field: "",
		Kind:  mainKind,
	}
	if mainKind == reflect.Struct {
		rootKey := key
		if !strings.HasSuffix(rootKey, "/") {
			rootKey += "/"
		}
		size := src.NumField()
		for i := 0; i < size; i++ {
			f := src.Type().Field(i)
			fKind := f.Type.Kind()
			if t := f.Tag.Get(tagName); t != "" {
				jsonTag := getFiledTag("json", &f)
				kvTag := getFiledTag(tagName, &f)
				if strings.HasPrefix(kvTag, "/") {
					keyTypeMap[kvTag] = fieldKind{
						Field: jsonTag,
						Kind:  fKind,
					}
				} else if strings.Contains(kvTag, "/") {
					keyTypeMap[rootKey+kvTag] = fieldKind{
						Field: jsonTag,
						Kind:  fKind,
					}
				}
			}
		}
	}
	return
}
