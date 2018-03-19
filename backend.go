package objectbind

import (
	"context"
	"net/url"

	"github.com/ti/objectbind/etcd"

	"github.com/ti/objectbind/file"
)

// NewBackend new backend
type NewBackend func(ctx context.Context, uri *url.URL) (Backend, error)

var backends = make(map[string]NewBackend)

const (
	schemeFile = "file"
	schemeEtcd = "etcd"
)

//SetBackend set backed
func SetBackend(scheme string, backend NewBackend) {
	backends[scheme] = backend
}

// Backend to support additional backends, such as etcd, consul, file ...
type Backend interface {
	Load(ctx context.Context, path string) (data map[string][]byte, err error)
	Save(ctx context.Context, path string, data []byte) error
	Watch(ctx context.Context, paths []string, onChange func(map[string][]byte)) error
}

func init() {
	SetBackend(schemeFile, func(ctx context.Context, uri *url.URL) (Backend, error) {
		return file.New(ctx, uri)
	})
	SetBackend(schemeEtcd, func(ctx context.Context, uri *url.URL) (Backend, error) {
		return etcd.New(etcd.WithContext(ctx), etcd.WithURI(uri))
	})
}
