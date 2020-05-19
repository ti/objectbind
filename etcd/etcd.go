package etcd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.etcd.io/etcd/v3/clientv3"
	"go.etcd.io/etcd/v3/pkg/transport"
)

// Etcd the default etcd backend
type Etcd struct {
	client *clientv3.Client
	locker sync.Locker
}

// Options the Etcd Options
type Options struct {
	ctx    context.Context
	Client *clientv3.Client
	URI    *url.URL
}

// Option how to set the Options
type Option func(*Options)

// WithEtcdClient with etcd client
func WithEtcdClient(client *clientv3.Client) Option {
	return func(o *Options) {
		o.Client = client
	}
}

// WithURI with etcd client
func WithURI(uri *url.URL) Option {
	return func(o *Options) {
		o.URI = uri
	}
}

// WithURIString with etcd client
func WithURIString(uri string) Option {
	return func(o *Options) {
		u, err := url.Parse(uri)
		if err != nil {
			panic(fmt.Errorf("parse etcd uri %s error for %s", uri, err))
		}
		o.URI = u
	}
}

// WithContext with context when etcd init
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.ctx = ctx
	}
}

// New etcd form custom client
func New(opts ...Option) (*Etcd, error) {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	if options.Client == nil {
		if options.URI == nil {
			return nil, errors.New("etcd client or etcd uri equired")
		}
		var err error
		options.Client, err = newEtcdClient(context.Background(), options.URI)
		if err != nil {
			return nil, err
		}
	}

	etcd := &Etcd{
		client: options.Client,
		locker: &sync.Mutex{},
	}
	return etcd, nil
}

// Load load data from path
func (e *Etcd) Load(ctx context.Context, path string) (map[string][]byte, error) {
	var opts []clientv3.OpOption
	if strings.HasSuffix(path, "/") {
		opts = append(opts, clientv3.WithPrefix())
	}
	getResp, err := e.client.Get(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	if len(getResp.Kvs) == 0 {
		return nil, nil
	}
	data := make(map[string][]byte)
	for _, kv := range getResp.Kvs {
		data[string(kv.Key)] = kv.Value
	}
	return data, nil
}

// Save save the data to path
func (e *Etcd) Save(ctx context.Context, path string, data []byte) (err error) {
	if len(data) > 0 {
		_, err = e.client.Put(ctx, path, string(data))
	} else {
		_, err = e.client.Delete(ctx, path)
	}
	return
}

// Watch watch the path
func (e *Etcd) Watch(ctx context.Context, paths []string, onChange func(data map[string][]byte)) error {
	paths = commonPaths(paths)
	for _, key := range paths {
		var opts []clientv3.OpOption
		if strings.HasSuffix(key, "/") {
			opts = append(opts, clientv3.WithPrefix())
		}
		rch := e.client.Watch(ctx, key, opts...)
		go func() {
			for wresp := range rch {
				var isChange bool
				for _, ev := range wresp.Events {
					if ev.Type == clientv3.EventTypePut || ev.Type == clientv3.EventTypeDelete {
						isChange = true
					}
				}
				if isChange {
					data := make(map[string][]byte)
					for _, evt := range wresp.Events {
						data[string(evt.Kv.Key)] = evt.Kv.Value
					}
					if len(data) > 0 {
						onChange(data)
					}
				}
			}
		}()
	}
	return nil
}

func newEtcdClient(ctx context.Context, etcdUri *url.URL) (*clientv3.Client, error) {
	etcdConfig := clientv3.Config{
		Endpoints:   strings.Split(etcdUri.Host, ","),
		DialTimeout: 30 * time.Second,
		Context:     ctx,
	}
	if etcdUri.User != nil && etcdUri.User.Username() != "" {
		etcdConfig.Username = etcdUri.User.Username()
		etcdConfig.Password, _ = etcdUri.User.Password()
	}
	etcdUriQuery := etcdUri.Query()
	cert := etcdUriQuery.Get("cert")
	if cert != "" {
		key := etcdUriQuery.Get("key")
		ca := etcdUriQuery.Get("ca")
		tlsInfo := transport.TLSInfo{
			CertFile:      cert,
			KeyFile:       key,
			TrustedCAFile: ca,
		}
		tlsConfig, err := tlsInfo.ClientConfig()
		if err != nil {
			return nil, err
		}
		etcdConfig.TLS = tlsConfig
	}
	return clientv3.New(etcdConfig)
}

func commonPaths(src []string) (dist []string) {
	commonPaths := make(map[string]bool)
	for _, v := range src {
		commonPaths[v] = true
	}
	for _, v := range src {
		for _, vv := range src {
			if strings.HasPrefix(vv, v) && len(v) != len(vv) {
				commonPaths[v] = true
				commonPaths[vv] = false
			}
		}
	}
	for k, v := range commonPaths {
		if v {
			dist = append(dist, k)
		}
	}
	return
}
