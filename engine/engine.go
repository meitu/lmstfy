package engine

import (
	"io"

	"github.com/bitleak/lmstfy/config"
)

type Engine interface {
	Publish(namespace, queue string, body []byte, ttlSecond, delaySecond uint32, tries uint16, priority uint8) (jobID string, err error)
	Consume(namespace, queue string, ttrSecond, timeoutSecond uint32) (job Job, err error)
	ConsumeMulti(namespace string, queues []string, ttrSecond, timeoutSecond uint32) (job Job, err error)
	ConsumeMultiWithFrozenTries(namespace string, queues []string, ttrSecond, timeoutSecond uint32) (job Job, err error)
	BatchConsume(namespace, queue string, count, ttrSecond, timeoutSecond uint32) (jobs []Job, err error)
	Delete(namespace, queue, jobID string) error
	Peek(namespace, queue, optionalJobID string) (job Job, err error)
	Size(namespace, queue string) (size int64, err error)
	Destroy(namespace, queue string) (count int64, err error)

	// Dead letter
	PeekDeadLetter(namespace, queue string) (size int64, jobID string, err error)
	DeleteDeadLetter(namespace, queue string, limit int64) (count int64, err error)
	RespawnDeadLetter(namespace, queue string, limit, ttlSecond int64) (count int64, err error)
	SizeOfDeadLetter(namespace, queue string) (size int64, err error)

	Shutdown()

	DumpInfo(output io.Writer) error
}

type EnginePool map[string]Engine

var engines = make(map[string]EnginePool)

func GetEngineByKind(kind, pool string) Engine {
	if pool == "" {
		pool = config.DefaultPoolName
	}
	k := engines[kind]
	if k == nil {
		return nil
	}
	return k[pool]
}

func GetPoolsByKind(kind string) []string {
	v, ok := engines[kind]
	if !ok {
		return []string{}
	}
	pools := make([]string, 0)
	for pool := range v {
		pools = append(pools, pool)
	}
	return pools
}

func ExistsPool(pool string) bool {
	if pool == "" {
		pool = config.DefaultPoolName
	}
	return GetEngine(pool) != nil
}

func ListKinds() []string {
	kinds := make([]string, 0)
	for name := range engines {
		kinds = append(kinds, name)
	}
	return kinds
}

func GetEngine(pool string) Engine {
	if pool == "" {
		pool = config.DefaultPoolName
	}
	allowKinds := ListKinds()
	for _, kind := range allowKinds {
		if e := GetEngineByKind(kind, pool); e != nil {
			return e
		}
	}
	return nil
}

func Register(kind, pool string, e Engine) {
	if p, ok := engines[kind]; ok {
		p[pool] = e
	} else {
		p = make(EnginePool)
		p[pool] = e
		engines[kind] = p
	}
}

func Shutdown() {
	for _, enginePool := range engines {
		for _, engine := range enginePool {
			engine.Shutdown()
		}
	}
}
