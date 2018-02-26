package cluster

type Task struct {
	Tasks []Tasker
}

func (p *Task) Start() {
	for _, runner := range p.Tasks {
		runner.Start()
	}
}

func (p *Task) Stop() {
	for _, runner := range p.Tasks {
		runner.Stop()
	}
}
