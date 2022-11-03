package ActorSystem

import (
    "errors"
    "sync"
    "sync/atomic"
)

type MessageExecActor struct {
    id        int32
    messages  chan Message
    config    *Config
    closeFlag int32
    wg        *sync.WaitGroup
    stat      *SystemStat
}

func (a *MessageExecActor) Receive(message Message) error {
    if atomic.LoadInt32(&a.closeFlag) == 2 {
        return errors.New("message actor is closed")
    }
    sz := len(a.messages)
    if sz >= a.config.ActorQueueSize {
        return errors.New("message actor is full")
    }
    a.messages <- message
    return nil
}
func (a *MessageExecActor) fire(message Message) {
    defer func() {
        recover()
        a.wg.Done()
        atomic.AddInt64(&a.stat.Running, -1)
    }()
    a.wg.Add(1)
    atomic.AddInt64(&a.stat.Running, 1)
    atomic.AddUint64(&a.stat.Finished, 1)

    message.Execute()
}
func (a *MessageExecActor) Start() {
    for message := range a.messages {
        a.fire(message)
    }
}
func (a *MessageExecActor) Stop() {
    if atomic.CompareAndSwapInt32(&a.closeFlag, 1, 2) {
        close(a.messages)
    }
}
func (a *MessageExecActor) StopIfFree() bool {
    sz := len(a.messages)
    if sz > 0 {
        return false
    }
    a.Stop()
    return true
}

var actorId int32

func NewActor(stat *SystemStat, wg *sync.WaitGroup, config *Config) Actor {
    actor := &MessageExecActor{
        id:       atomic.AddInt32(&actorId, 1),
        messages: make(chan Message, config.ActorQueueSize),
        config:   config,
        wg:       wg,
        stat:     stat,
    }
    go actor.Start()
    return actor
}
