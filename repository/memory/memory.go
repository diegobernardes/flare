package memory

type errMemory struct {
	message       string
	alreadyExists bool
	pathConflict  bool
	notFound      bool
}

func (e *errMemory) Error() string       { return e.message }
func (e *errMemory) AlreadyExists() bool { return e.alreadyExists }
func (e *errMemory) PathConflict() bool  { return e.pathConflict }
func (e *errMemory) NotFound() bool      { return e.notFound }
