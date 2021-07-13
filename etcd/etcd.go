package etcd

import (
	"context"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/ti/objectbind"

	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func init() {
	objectbind.SetBackend("etcd", func(ctx context.Context, uri *url.URL) (objectbind.Backend, error) {
		return New(ctx, uri)
	})
}

// Etcd the etcd client
type Etcd struct {
	client *clientv3.Client
	Root   string
}

// New new etcd client
func New(ctx context.Context, uri *url.URL) (*Etcd, error) {
	etcdConfig := clientv3.Config{
		Endpoints:   strings.Split(uri.Host, ","),
		DialTimeout: 10 * time.Second,
		Context:     ctx,
	}
	if uri.User != nil && uri.User.Username() != "" {
		etcdConfig.Username = uri.User.Username()
		etcdConfig.Password, _ = uri.User.Password()
	}
	query := uri.Query()
	if cert := query.Get("cert"); cert != "" {
		key := query.Get("key")
		ca := query.Get("ca")
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
	cli, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, err
	}
	_, err = cli.Status(ctx, uri.Host)
	if err != nil {
		return nil, err
	}
	return &Etcd{
		client: cli,
		Root:   filepath.Dir(uri.Path) + "/",
	}, nil
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
		watchChan := e.client.Watch(ctx, key, opts...)
		go watch(watchChan, onChange)
	}
	return nil
}

func watch(watchChan clientv3.WatchChan, onChange func(data map[string][]byte)) {
	for watchResponse := range watchChan {
		var isChange bool
		for _, ev := range watchResponse.Events {
			if ev.Type == clientv3.EventTypePut || ev.Type == clientv3.EventTypeDelete {
				isChange = true
			}
		}
		if isChange {
			data := make(map[string][]byte)
			for _, evt := range watchResponse.Events {
				data[string(evt.Kv.Key)] = evt.Kv.Value
			}
			if len(data) > 0 {
				onChange(data)
			}
		}
	}
}

// Close close the etcd
func (e *Etcd) Close(_ context.Context) error {
	return e.client.Close()
}

// Client get the etcd client
func (e *Etcd) Client() *clientv3.Client {
	return e.client
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
