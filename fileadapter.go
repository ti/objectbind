package objectbind

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func (b *Binder) saveJSONFile(ctx context.Context, path string, data []byte) (err error) {
	data, err = b.json2Codec(path, data)
	if err != nil {
		return fmt.Errorf("objectbind.JSON2Codec %s error for %s", path, err)
	}
	return b.backend.Save(ctx, b.getFileName(path), data)
}

func (b *Binder) loadJSONFile(ctx context.Context, path string) ([]*mapData, error) {
	field := b.fields[path]
	if field == nil {
		return nil, errors.New("can not get path not in fields - " + path)
	}
	files, err := b.backend.Load(ctx, b.getFileName(path))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}
	if !strings.HasSuffix(path, "/") {
		d, ok := files[b.getFileName(path)]
		if !ok || len(d) < 1 {
			return nil, nil
		}
		jsonData, err := b.codec2JSON(path, d)
		if err != nil {
			return nil, err
		}
		return []*mapData{{
			Key:   path,
			Value: string(jsonData),
		},
		}, nil
	}
	var keys []string
	for k := range files {
		_, item := filepath.Split(k)
		k = b.getName(item)
		if k != "" {
			keys = append(keys, k)
		}
	}
	if field.Kind == reflect.Slice {
		stringIntSort(keys)
	} else {
		sort.Strings(keys)
	}
	var dist []*mapData
	for _, k := range keys {
		d, ok := files[path+b.getFileName(k)]
		if !ok || len(d) < 1 {
			return nil, nil
		}
		jsonData, err := b.codec2JSON(path, d)
		if err != nil {
			return nil, err
		}

		dist = append(dist, &mapData{
			Key:   path + k,
			Value: string(jsonData),
		})
	}
	return dist, nil
}

func (b *Binder) watchJSONFile(ctx context.Context, paths []string, onChange func([]*mapData)) error {
	for i, v := range paths {
		paths[i] = b.getFileName(v)
	}
	return b.backend.Watch(ctx, paths, func(m map[string][]byte) {
		var data []*mapData
		for k, v := range m {
			filename := b.getName(k)
			if filename != "" && !strings.HasSuffix(filename, "/") {
				jsonData, err := b.codec2JSON(filename, v)
				if err != nil {
					logrus.WithField("action", "objectbind.Watch.Load").WithField("path", filename).Error(err)
				}
				data = append(data, &mapData{
					Key:   filename,
					Value: string(jsonData),
				})
			}
		}
		onChange(data)
	})
}

func stringIntSort(src []string) {
	data := make([]int, len(src))
	for i, v := range src {
		n, err := strconv.Atoi(v)
		if err != nil {
			sort.Strings(src)
			return
		}
		data[i] = n
	}
	sort.Ints(data)
	for i, v := range data {
		src[i] = strconv.Itoa(v)
	}
}

func (b *Binder) getFileName(key string) string {
	if !strings.HasSuffix(key, "/") && b.withExtension {
		key += b.extension
	}
	return key
}

func (b *Binder) getName(filename string) string {
	if b.withExtension {
		ext := filepath.Ext(filename)
		if ext == b.extension {
			filename = filename[0 : len(filename)-b.lenExtension]
		} else {
			filename = ""
		}
	}
	return filename
}
