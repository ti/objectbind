package file

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

// File the file watcher
type File struct {
	watcher         *fsnotify.Watcher
	wd              string
	wdLen           int
	basedOnRootPath bool
}

// New new file client
func New(ctx context.Context, u *url.URL) (*File, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	fb := &File{}
	path := u.Host + u.Path
	if strings.HasPrefix(path, "/") {
		fb.basedOnRootPath = true
	}
	if !strings.HasSuffix(wd, "/") {
		wd += "/"
	}
	fb.wd = wd
	fb.wdLen = len(wd)
	return fb, nil
}

func (f *File) getPath(p string) string {
	if f.basedOnRootPath || !strings.HasPrefix(p, "/") {
		return p
	}
	if strings.HasPrefix(p, f.wd) {
		return p[f.wdLen:]
	}
	return p
}

// Load load data from path
func (f *File) Load(ctx context.Context, path string) (data map[string][]byte, err error) {
	data = map[string][]byte{}
	var fileData []byte
	if !strings.HasSuffix(path, "/") {
		fileData, err = ioutil.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				err = nil
				return
			}
			return
		}
		data[path] = fileData
		return
	}
	var infos []os.FileInfo
	infos, err = ioutil.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			return
		}
		return
	}

	if len(infos) == 0 {
		return
	}

	for _, v := range infos {
		if v.IsDir() || strings.HasPrefix(v.Name(), ".") {
			continue
		}
		filePath := path + v.Name()
		fileData, err = ioutil.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
		}
		data[filePath] = fileData
	}
	return
}

// Save save data to path
func (f *File) Save(_ context.Context, path string, data []byte) error {
	if len(data) > 0 {
		return writeFile(path, data)
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func writeFile(path string, data []byte) error {
	err := ioutil.WriteFile(path, data, 0700)
	if os.IsNotExist(err) {
		fileDir := filepath.Dir(path)
		if _, pathStatErr := os.Stat(fileDir); pathStatErr != nil {
			if !os.IsNotExist(pathStatErr) {
				return fmt.Errorf("try to open file error %s, try to stat dir %s,  error %s", err, fileDir, pathStatErr)
			}
			if mkdirError := os.MkdirAll(fileDir, os.FileMode(0700)); mkdirError != nil {
				return fmt.Errorf("try to open file %s, try to mkdir %s,  error %s", err, path, mkdirError)
			}
		}
		err = ioutil.WriteFile(path, data, 0700)
	}
	return err
}

// Watch watch the path
func (f *File) Watch(ctx context.Context, paths []string, onChange func(map[string][]byte)) (err error) {
	if f.watcher == nil {
		f.watcher, err = fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("fsnotify.NewWatcher error for %s", err)
		}
	}
	for _, v := range paths {
		if v == "" {
			v = "./"
		}
		err = f.watcher.Add(v)
		if !strings.HasSuffix(v, "/") {
			if os.IsNotExist(err) {
				v, _ = filepath.Split(v)
				err = f.watcher.Add(v)
			}
		}
		if err != nil {
			if os.IsNotExist(err) && strings.HasSuffix(v, "/") {
				err = os.MkdirAll(v, os.FileMode(0700))
				if err == nil {
					err = f.watcher.Add(v)
				}
			}
		}

		if err != nil {
			return fmt.Errorf("fsnotify.Add %s error for %s", v, err)
		}
	}
	go f.watch(onChange)
	return err
}

func (f *File) watch(onChange func(map[string][]byte)) {
	var err error
	defer func() {
		err = f.watcher.Close()
		if err != nil {
			logrus.WithField("action", "fsnotify.Close").Error(err)
		}
	}()
	for {
		select {
		case event, ok := <-f.watcher.Events:
			if !ok {
				return
			}
			if event.Op == fsnotify.Chmod || strings.HasSuffix(event.Name, "~") {
				continue
			}
			_, filename := filepath.Split(event.Name)
			if strings.HasPrefix(filename, ".") {
				continue
			}
			path := event.Name
			fileData, err := ioutil.ReadFile(path)
			if err != nil && !os.IsNotExist(err) {
				logrus.WithField("action", "read_file").WithField("path", path).Error(err)
			}
			data := make(map[string][]byte)
			filePath := f.getPath(path)
			data[filePath] = fileData
			onChange(data)

		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}
			logrus.WithField("action", "fsnotify.Watcher").Error(err)
		}
	}
}
