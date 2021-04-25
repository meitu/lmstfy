package migration

import (
	"io"

	"github.com/bitleak/lmstfy/engine"
)

type Engine struct {
	oldEngine engine.Engine
	newEngine engine.Engine
}

func NewEngine(old, new engine.Engine) engine.Engine {
	return &Engine{
		oldEngine: old,
		newEngine: new,
	}
}

func (e *Engine) Queue(meta engine.QueueMeta) (engine.Queue, error) {
	return Queue{
		meta: meta,
		e:    e,
	}, nil
}

func (e *Engine) Queues(metas []engine.QueueMeta) (engine.Queues, error) {
	return Queues{
		meta: metas,
		e:    e,
	}, nil
}

func (e *Engine) DeadLetter(meta engine.QueueMeta) (engine.DeadLetter, error) {
	return DeadLetter{
		meta: meta,
		e:    e,
	}, nil
}

func (e *Engine) Shutdown() {
	e.oldEngine.Shutdown()
	e.newEngine.Shutdown()
}

func (e *Engine) DumpInfo(output io.Writer) error {
	return e.newEngine.DumpInfo(output)
}
