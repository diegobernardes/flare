package cluster

import "github.com/go-kit/kit/log"

type GroupTask struct {
	Tasks  []Tasker
	Logger log.Logger
}

func (gt *GroupTask) Start() {
	for _, tt := range gt.Tasks {
		tt.Start()
	}
}

func (gt *GroupTask) Stop() {
	for _, tt := range gt.Tasks {
		tt.Stop()
	}
}
