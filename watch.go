package objectbind

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
)

// watch save the data
func (b *Binder) watch(ctx context.Context) error {
	var watchFiles []string
	for _, v := range b.fields {
		watchFiles = append(watchFiles, v.Path)
	}
	return b.watchJSONFile(ctx, watchFiles, func(kvs []*mapData) {
		b.locker.Lock()
		defer b.locker.Unlock()
		loadedPaths := make(map[string]bool)
		var dataFiles []*mapData
		var changedPaths []string
		for _, kv := range kvs {
			currentKV, ok := b.currentFiles[kv.Key]
			if ok && currentKV.Value == kv.Value {
				continue
			}
			field, ok := b.fields[kv.Key]
			if !ok {
				field, ok = b.fields[filepath.Dir(kv.Key)+"/"]
			}
			if !ok || loadedPaths[field.Path] {
				continue
			}
			if !strings.HasSuffix(field.Path, "/") {
				dataFiles = append(dataFiles, kv)
				loadedPaths[kv.Key] = true
				continue
			}
			changedFiles, err := b.loadJSONFile(ctx, field.Path)
			if err != nil {
				logrus.WithField("action", "objectbind.LoadFile").WithField("path", field.Path).Error(err)
				return
			}
			dataFiles = append(dataFiles, changedFiles...)
			loadedPaths[field.Path] = true
			changedPaths = append(changedPaths, field.Path)
		}
		if len(dataFiles) == 0 {
			return
		}
		err := unmarshal(b.root, dataFiles, b.instance, b.tagName)
		if err != nil {
			logrus.WithField("action", "objectbind.onChange.Unmarshal").Error(err)
		}
		currentFiles := b.currentFiles
		for _, v := range changedPaths {
			for k := range currentFiles {
				path, _ := filepath.Split(k)
				if v == path {
					delete(b.currentFiles, k)
				}
			}
		}
		for _, kv := range dataFiles {
			b.currentFiles[kv.Key] = kv
		}

		b.notifyChanges(ctx)
	})
}

//notifyChanges notify some trigger on data
func (b *Binder) notifyChanges(ctx context.Context) {
	if reflect.DeepEqual(b.instance, b.preInstance) {
		return
	}
	for _, t := range b.triggers {
		oldValue, _ := getFieldValue(b.preInstance, t.filed, true, false)
		newValue, err := getFieldValue(b.instance, t.filed, true, false)
		if err != nil {
			logrus.Warnf("can not get value by field %s for %s", t.filed, err)
			continue
		}
		if newValue == nil && oldValue == nil {
			continue
		}
		if newValue == nil {
			tx := reflect.Indirect(reflect.ValueOf(oldValue)).Type()
			newValue = reflect.New(tx).Interface()
			if reflect.ValueOf(oldValue).Kind() != reflect.Ptr {
				newValue = reflect.Indirect(reflect.ValueOf(newValue)).Interface()
			}
		}
		if oldValue == nil {
			tx := reflect.Indirect(reflect.ValueOf(newValue)).Type()
			oldValue = reflect.New(tx).Interface()
			if reflect.ValueOf(newValue).Kind() != reflect.Ptr {
				oldValue = reflect.Indirect(reflect.ValueOf(oldValue)).Interface()
			}
		}
		if reflect.DeepEqual(newValue, oldValue) {
			continue
		}
		t.callback(newValue, oldValue)
	}
	b.preInstance = clone(b.instance, false)
}
