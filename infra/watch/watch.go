package watch

// OpOption, vamos ter isso? tipo prefix, etc..?
type Watcher interface {
	// Watch(ctx context.Context, key string, opts ...OpOption) WatchChan
	Close() error
}
