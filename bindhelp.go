package objectbind

import (
	"context"
)

func (b *Binder) loadFiles(ctx context.Context) ([]*mapData, error) {
	var files []*mapData
	for _, v := range b.fields {
		fs, err := b.loadJSONFile(ctx, v.Path)
		if err != nil {
			return nil, err
		}
		if len(fs) > 0 {
			files = append(files, fs...)
		}
	}
	return files, nil
}

func (b *Binder) saveCurrentDataWithoutCompare(ctx context.Context) error {
	data, err := marshal(b.root, b.instance, b.tagName)
	if err != nil {
		return err
	}
	for _, v := range data {
		if err := b.saveJSONFile(ctx, v.Key, []byte(v.Value)); err != nil {
			return err
		}
	}
	b.save2CurrentFiles(data)
	return nil
}

func (b *Binder) save2CurrentFiles(files []*mapData) {
	b.currentFiles = make(map[string]*mapData)
	for _, v := range files {
		b.currentFiles[v.Key] = v
	}
}
