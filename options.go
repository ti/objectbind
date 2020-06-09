package objectbind

import (
	"sync"
	"time"
)

//Options the Options of config
type Options struct {
	backend          Backend
	locker           sync.Locker
	codec            Codec
	tagName          string
	withoutExtension bool
	withoutWatch     bool
	ttl              time.Duration
}

//Option is just Option functions
type Option func(*Options)

// WithBackend set default config of the instance
func WithBackend(backend Backend) Option {
	return func(o *Options) {
		o.backend = backend
	}
}

//WithLocker load timeout
func WithLocker(l sync.Locker) Option {
	return func(o *Options) {
		o.locker = l
	}
}

//WithCodec with codec, default is json
func WithCodec(c Codec) Option {
	return func(o *Options) {
		o.codec = c
	}
}

//WithTagName custom tag name for your binder, default is bind
func WithTagName(s string) Option {
	return func(o *Options) {
		o.tagName = s
	}
}

//WithoutExtension no extension in write file, default is your file extension or as same as your codecs name
func WithoutExtension(e bool) Option {
	return func(o *Options) {
		o.withoutExtension = e
	}
}

//WithoutWatch not watch the files changes
func WithoutWatch(w bool) Option {
	return func(o *Options) {
		o.withoutWatch = w
	}
}

// WithTTL Do Reload in a ttl loop.
func WithTTL(t time.Duration) Option {
	return func(o *Options) {
		o.ttl = t
	}
}
