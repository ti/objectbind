package objectbind

import (
	"context"
	"errors"
	"fmt"

	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

// Binder the binder instance
type Binder struct {
	backend  Backend
	codec    Codec
	locker   sync.Locker
	instance interface{}
	triggers []*trigger
	tagName  string

	// files
	root         string
	rootDir      string
	fields       map[string]*field
	currentFiles map[string]*mapData

	preInstance interface{}

	withExtension bool
	extension     string
	lenExtension  int
}

type trigger struct {
	filed    string
	callback func(value, preValue interface{})
}

// Bind bind target to uri
func Bind(ctx context.Context, target interface{}, uri string, opts ...Option) (*Binder, error) {
	var opt = &Options{}
	for _, o := range opts {
		o(opt)
	}
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil, errors.New("target value is not a pointer")
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u.Scheme = schemeFile
	}
	if opt.backend == nil {
		backend, ok := backends[u.Scheme]
		if !ok {
			return nil, fmt.Errorf("%s for bind is not supported", u.Scheme)
		}
		opt.backend, err = backend(ctx, u)
		if err != nil {
			return nil, err
		}
	}
	ext := filepath.Ext(u.Path)
	if opt.codec == nil {
		if c, ok := defaultCodes[ext]; ok {
			opt.codec = c
		} else {
			opt.codec = defaultCodes[".json"]
		}
	}
	root := u.Path
	if !opt.withoutExtension {
		if ext == "" {
			ext = "." + opt.codec.String()
		}
		root = strings.TrimSuffix(root, ext)
	}
	if opt.locker == nil {
		opt.locker = &sync.Mutex{}
	}
	if opt.tagName == "" {
		opt.tagName = "bind"
	}
	binder := &Binder{
		locker:        opt.locker,
		instance:      target,
		backend:       opt.backend,
		root:          root,
		codec:         opt.codec,
		withExtension: !opt.withoutExtension,
		extension:     ext,
		lenExtension:  len(ext),
		tagName:       opt.tagName,
	}
	err = binder.init(ctx, opt)
	return binder, err
}

// BindField bind field
func (b *Binder) init(ctx context.Context, opt *Options) (err error) {
	if strings.HasSuffix(b.root, "/") {
		b.rootDir = b.root
	} else {
		i := strings.LastIndex(b.root, "/")
		if i <= 0 {
			b.rootDir = ""
		} else {
			b.rootDir = b.root[:i+1]
		}
	}
	b.fields = getFields(b.root, b.instance, b.tagName)
	// load the files, check if the files exist
	files, errLoadFile := b.loadFiles(ctx)
	if errLoadFile != nil {
		return fmt.Errorf("load all files error for %s", errLoadFile)
	}
	if len(files) == 0 {
		err = b.saveCurrentDataWithoutCompare(ctx)
	} else {
		err = unmarshal(b.root, files, b.instance, b.tagName)
		b.save2CurrentFiles(files)
	}
	if err != nil {
		return
	}
	b.preInstance = clone(b.instance, false)
	if !opt.withoutWatch {
		err = b.watch(ctx)
	}
	return
}

// BindField bind field
func (b *Binder) BindField(field string, onValue func(value, preValue interface{})) {
	b.locker.Lock()
	defer b.locker.Unlock()
	v, err := getFieldValue(b.instance, field, true, false)
	if err != nil {
		panic(fmt.Errorf("bind field %s error for %s", field, err))
	}
	onValue(v, v)
	tg := &trigger{
		filed:    field,
		callback: onValue,
	}
	b.triggers = append(b.triggers, tg)
}

// ForceLoad force load form backend
func (b *Binder) ForceLoad(ctx context.Context) error {
	files, err := b.loadFiles(ctx)
	if err != nil {
		return fmt.Errorf("load all files error for %s", err)
	}
	if len(files) == 0 {
		return ErrNoFiles
	}
	err = unmarshal(b.root, files, b.instance, b.tagName)
	if err != nil {
		return err
	}
	b.notifyChanges(ctx)
	return nil
}

// Save save the data
func (b *Binder) Save(ctx context.Context) error {
	memeryData, err := marshal(b.root, b.instance, b.tagName)
	if err != nil {
		return err
	}
	// compare
	var todoSave []*mapData
	for _, memeryItem := range memeryData {
		currentItem, ok := b.currentFiles[memeryItem.Key]
		if !ok || memeryItem.Value != currentItem.Value {
			todoSave = append(todoSave, memeryItem)
		}
	}
	memeryDataMap := make(map[string]*mapData)
	for _, v := range memeryData {
		memeryDataMap[v.Key] = v
	}
	for k := range b.currentFiles {
		memItem, ok := memeryDataMap[k]
		if !ok || memItem.Value == "" {
			todoSave = append(todoSave, &mapData{
				Key:   k,
				Value: "",
			})
		}
	}
	for _, v := range todoSave {
		if v.Value == "null" || strings.HasSuffix(v.Key, "/") {
			continue
		}
		if err := b.saveJSONFile(ctx, v.Key, []byte(v.Value)); err != nil {
			return err
		}
	}
	return nil
}
