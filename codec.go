package objectbind

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"strings"
)

// Codec the codes interface for you can custom your codec
type Codec interface {
	String() string
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

var defaultCodes = map[string]Codec{
	".json": &jsonCodec{},
	".yaml": &yamlCodec{},
}

func (b *Binder) json2Codec(path string, src []byte) ([]byte, error) {
	if b.codec == nil {
		return src, nil
	}
	dir, _ := filepath.Split(path)
	field, ok := b.fields[dir]
	if !ok {
		field, ok = b.fields[path]
	}
	if !ok {
		var fields []string
		for k := range b.fields {
			fields = append(fields, k)
		}
		return nil, fmt.Errorf("no field %s found in %s", path, fields)
	}
	isDIR := strings.HasSuffix(field.Path, "/")
	var data interface{}
	if !isDIR {
		data = newWithValue(field.NullValue, src, false)
	} else {
		data = newWithValue(field.ChildNullValue, src, false)
	}
	if data == nil {
		return nil, nil
	}
	return b.codec.Marshal(data)
}

func (b *Binder) codec2JSON(path string, src []byte) ([]byte, error) {
	if b.codec == nil {
		return src, nil
	}

	var dir string
	if !strings.HasSuffix(path, "/") {
		dir, _ = filepath.Split(path)
	}
	field, ok := b.fields[dir]
	if !ok {
		field, ok = b.fields[path]
	}
	if !ok {
		var fields []string
		for k := range b.fields {
			fields = append(fields, k)
		}
		return nil, fmt.Errorf("can not found field %s in %s", path, fields)
	}
	isDIR := strings.HasSuffix(field.Path, "/")
	var data interface{}
	if !isDIR {
		data = newWithValue(field.NullValue, nil, true)
	} else {
		data = newWithValue(field.ChildNullValue, nil, true)
	}
	err := b.codec.Unmarshal(src, data)
	if err != nil {
		return nil, fmt.Errorf("%s unmarshal path %s 's data %s error for %s", b.codec.String(), path, string(src), err)
	}
	needRootUnmarshal := path == b.root && len(b.fields) > 1
	if needRootUnmarshal {
		data = convertMainData(data, b.tagName)
	}
	return json.Marshal(data)
}

type jsonCodec struct{}

func (j *jsonCodec) String() string {
	return "json"
}

func (j *jsonCodec) Marshal(v interface{}) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return nil, err
	}
	b = removeEndLinesBy(b, removePreLine, []byte(": null\n"), []byte(": null,\n"))
	return b, nil
}

func (j *jsonCodec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

type yamlCodec struct{}

func (j *yamlCodec) String() string {
	return "yaml"
}

func (j *yamlCodec) Unmarshal(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}

func (j *yamlCodec) Marshal(v interface{}) ([]byte, error) {
	b, err := yaml.Marshal(v)
	data := removeEndLinesBy(b, nil, []byte(": []\n"), []byte(": {}\n"))
	return data, err
}

func removeEndLinesBy(data []byte, removePreLine func(preLine []byte) []byte, ends ...[]byte) []byte {
	var lineBreak = byte('\n')
	reader := bufio.NewReader(bytes.NewBuffer(data))
	var ret []byte
	for {
		line, lineErr := reader.ReadBytes(lineBreak)
		if lineErr != nil {
			ret = append(ret, line...)
			break
		}
		var ignore bool
		for _, end := range ends {
			if bytes.HasSuffix(line, end) {
				ignore = true
				break
			}
		}
		if ignore {
			if removePreLine != nil {
				ret = removePreLine(ret)
			}
			continue
		}
		ret = append(ret, line...)
	}
	return ret
}

func removePreLine(ret []byte) []byte {
	if bytes.HasSuffix(ret, []byte(",\n")) {
		ret = ret[:len(ret)-2]
		ret = append(ret, '\n')
	}
	return ret
}
