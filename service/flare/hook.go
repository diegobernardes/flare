package flare

import "github.com/diegobernardes/flare"

type hook struct {
	resource *flare.HookResource
}

func (h *hook) init() {
	h.resource = &flare.HookResource{}
	// h.resource.RegisterDelete(callback func(context.Context, string) error)
}
