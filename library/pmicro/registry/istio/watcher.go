package istio

import (
	"errors"
	"go-micro.dev/v4/registry"
)

type istioWatcher struct {
}

func newIstioWatcher(opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	return &istioWatcher{}, nil
}

func (ew *istioWatcher) Next() (*registry.Result, error) {

	return nil, errors.New("could not get next")
}

func (ew *istioWatcher) Stop() {

}
