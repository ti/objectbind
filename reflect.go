package objectbind

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"reflect"
	"strconv"
	"strings"
)

// clone fully copy config instance, include map
func clone(src interface{}, forceAddr bool) interface{} {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	dec := json.NewDecoder(&buf)
	err := enc.Encode(src)
	if err != nil {
		return err
	}
	t := reflect.Indirect(reflect.ValueOf(src)).Type()
	dist := reflect.New(t).Interface()
	err = dec.Decode(dist)
	if err != nil {
		return err
	}
	rv := reflect.ValueOf(src)
	if forceAddr || rv.Kind() == reflect.Ptr {
		return dist
	}
	return reflect.Indirect(reflect.ValueOf(dist)).Interface()
}

// newWithValue fully copy config instance, include map
func newWithValue(src interface{}, v []byte, forceAddr bool) interface{} {
	t := reflect.Indirect(reflect.ValueOf(src)).Type()
	dist := reflect.New(t).Interface()
	if len(v) > 0 {
		buf := bytes.NewBuffer(v)
		dec := json.NewDecoder(buf)
		err := dec.Decode(dist)
		if err != nil {
			logrus.WithField("action", "objectbind.NewWithValue").Error(err)
		}
	}
	rv := reflect.ValueOf(src)
	if forceAddr || rv.Kind() == reflect.Ptr {
		return dist
	}
	return reflect.Indirect(reflect.ValueOf(dist)).Interface()
}

// getFieldValue get GetFieldValue
func getFieldValue(src interface{}, path string, newValueIfNil, forceAddr bool) (d interface{}, err error) {
	v := reflect.ValueOf(src)
	dist, orgKind, err := getFieldValueReflect(v, v.Kind(), compile(path), newValueIfNil)
	if err != nil {
		if err == errorOutOfRange {
			return nil, err
		}
		return nil, fmt.Errorf("path %s err for %s", path, err)
	}
	if !dist.IsValid() {
		return nil, fmt.Errorf("path %s is in invalid", path)
	}
	if forceAddr || orgKind == reflect.Ptr {
		return dist.Interface(), nil
	}
	return reflect.Indirect(dist).Interface(), nil
}

var errorOutOfRange = errors.New("out of range")

func getFieldValueReflect(src reflect.Value, orgKind reflect.Kind, paths []string, newValueIfNil bool) (reflect.Value, reflect.Kind, error) {
	if len(paths) == 0 {
		return src, orgKind, nil
	}
	if !src.IsValid() {
		return reflect.Value{}, reflect.Invalid, fmt.Errorf("%s is not valid", paths)
	}
	key := paths[0]

	switch k := src.Kind(); k {
	case reflect.Map:
		paths = paths[1:]
		srcTmp := src.MapIndex(reflect.ValueOf(key))
		if newValueIfNil && !srcTmp.IsValid() {
			el := reflect.TypeOf(src.Interface()).Elem()
			orgKind = el.Kind()
			src = reflect.New(el)
		} else {
			src = srcTmp
		}
	case reflect.Struct:
		paths = paths[1:]
		src = src.FieldByName(key)
	case reflect.Slice:
		paths = paths[1:]
		n, en := strconv.Atoi(key)
		if en != nil {
			return reflect.Value{}, reflect.Invalid, fmt.Errorf("%s is not a number %s", key, en)
		}
		if src.Len() < n+1 {
			if newValueIfNil {
				t := reflect.MakeSlice(reflect.TypeOf(src.Interface()), 1, 2)
				v := t.Index(0)
				orgKind = v.Kind()
				var tx reflect.Type
				if orgKind == reflect.Ptr {
					vt := v.Type()
					src = reflect.New(vt)
				} else {
					tx = reflect.Indirect(v).Type()
					src = reflect.New(tx)
				}
				return getFieldValueReflect(src, orgKind, paths, newValueIfNil)
			}
			return reflect.Value{}, reflect.Invalid, errorOutOfRange
		}
		srcTmp := src.Index(n)
		if newValueIfNil && !srcTmp.IsValid() {
			t := reflect.MakeSlice(reflect.TypeOf(src.Interface()), 1, 2)
			tx := reflect.Indirect(t.Index(0)).Type()
			src = reflect.New(tx)
		} else {
			src = srcTmp
		}
	case reflect.Ptr:
		orgKind = k
		src = reflect.Indirect(src)
	default:
		return reflect.Value{}, k, fmt.Errorf("%s is not supported", k)
	}
	return getFieldValueReflect(src, orgKind, paths, newValueIfNil)
}

func compile(src string) []string {
	var a []string
	var p int
	l := len(src)
	if l == 0 {
		return a
	}
	var v uint8
	for i := 0; i < l; i++ {
		v = src[i]
		if v == ']' {
			a = append(a, src[p:i])
			p = i + 2
			i++
			continue
		}
		if v == '.' || v == '[' {
			a = append(a, src[p:i])
			p = i + 1
		}
	}
	if p < l {
		a = append(a, src[p:])
	}
	return a
}

func getFields(root string, target interface{}, tagName string) (dist map[string]*field) {
	kind, i := getReflectInterface(target)
	childNullValue := getChildValueByNonPtrInterface(kind, i)
	mainField := &field{
		Path:           root,
		Field:          "",
		JsonTag:        "",
		NullValue:      target,
		ChildNullValue: childNullValue,
		Kind:           kind,
	}
	dist = map[string]*field{
		root: mainField,
	}
	if kind != reflect.Struct {
		return
	}
	var rootDir string
	if strings.HasSuffix(root, "/") {
		rootDir = root
	} else {
		i := strings.LastIndex(root, "/")
		if i <= 0 {
			rootDir = ""
		} else {
			rootDir = root[:i+1]
		}
	}
	src := reflect.ValueOf(i)
	size := src.NumField()
	for i := 0; i < size; i++ {
		f := src.Type().Field(i)
		if t := f.Tag.Get(tagName); t != "" {
			if !strings.HasPrefix(t, "/") {
				t = rootDir + t
			}
			tmp := src.Field(i)
			if tmp.Kind() == reflect.Ptr {
				tmp = reflect.ValueOf(reflect.Indirect(tmp))
			}
			fd := &field{
				Path:           t,
				Field:          f.Name,
				JsonTag:        getFiledTag("json", &f),
				NullValue:      clone(tmp.Interface(), true),
				Kind:           f.Type.Kind(),
				ChildNullValue: getChildValueByField(&f),
			}
			dist[t] = fd
		}
	}
	return
}

type field struct {
	Path           string
	Field          string
	JsonTag        string
	NullValue      interface{}
	ChildNullValue interface{}
	Kind           reflect.Kind
}

func getChildValueByNonPtrInterface(kind reflect.Kind, i interface{}) (data interface{}) {
	if kind == reflect.Slice {
		t := reflect.MakeSlice(reflect.TypeOf(i), 1, 2)
		tmpValue := t.Index(0)
		if tmpValue.Kind() == reflect.Ptr {
			tmpValue = reflect.ValueOf(reflect.Indirect(tmpValue))
		}
		data = reflect.New(tmpValue.Type()).Interface()
	} else if kind == reflect.Map {
		data = reflect.New(reflect.TypeOf(i)).Interface()
	}
	return
}

func getChildValueByField(f *reflect.StructField) (data interface{}) {
	fKind := f.Type.Kind()
	switch fKind {
	case reflect.Map:
		data = reflect.New(f.Type.Elem()).Interface()
	case reflect.Slice:
		t := reflect.MakeSlice(f.Type, 1, 2)
		tmpValue := t.Index(0)
		if tmpValue.Kind() == reflect.Ptr {
			tmpValue = reflect.ValueOf(reflect.Indirect(tmpValue))
		}
		data = reflect.New(tmpValue.Type()).Interface()
	}
	return
}
