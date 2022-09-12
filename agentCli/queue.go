package agentCli

import (
	"sync"
	"time"

	"op-agent/config"
)

var DiscoveryQueue map[string]*Queue
var dcLock sync.Mutex

func init() {
	DiscoveryQueue = make(map[string]*Queue)
}

// Queue contains information for managing discovery requests
type Queue struct {
	sync.Mutex

	name         string
	done         chan struct{}
	queue        chan AgentNode
	queuedKeys   map[AgentNode]time.Time
	consumedKeys map[AgentNode]time.Time
}

// CreateOrReturnQueue allows for creation of a new discovery queue or
// returning a pointer to an existing one given the name.
func CreateOrReturnQueue(name string) *Queue {
	dcLock.Lock()
	defer dcLock.Unlock()
	if q, found := DiscoveryQueue[name]; found {
		return q
	}

	q := &Queue{
		name:         name,
		queuedKeys:   make(map[AgentNode]time.Time),
		consumedKeys: make(map[AgentNode]time.Time),
		queue:        make(chan AgentNode, config.DiscoveryQueueCapacity),
	}

	DiscoveryQueue[name] = q
	return q
}

// Push enqueues a key if it is not on a queue and is not being
// processed; silently returns otherwise.
func (q *Queue) Push(key AgentNode) {
	q.Lock()
	defer q.Unlock()

	// is it enqueued already?
	if _, found := q.queuedKeys[key]; found {
		return
	}

	// is it being processed now?
	//if _, found := q.consumedKeys[key]; found {
	//	return
	//}

	q.queuedKeys[key] = time.Now()
	q.queue <- key
}

// QueueLen returns the length of the queue (channel size + queued size)
func (q *Queue) QueueLen() int {
	q.Lock()
	defer q.Unlock()
	return len(q.queue) + len(q.queuedKeys)
}

func (q *Queue) Consume() AgentNode {
	q.Lock()
	queue := q.queue
	q.Unlock()

	key := <-queue

	q.Lock()
	defer q.Unlock()

	//q.consumedKeys[key] = q.queuedKeys[key]

	delete(q.queuedKeys, key)

	return key
}

func (q *Queue) Release(key AgentNode) {
	q.Lock()
	defer q.Unlock()

	delete(q.consumedKeys, key)
}

