package actor

import (
    "errors"
    "sync"
    "sync/atomic"
)

type (
    actorContext struct {
        id       int
        messages chan Message
        actor    Actor
        config   *Config
        wg       *sync.WaitGroup
        stat     *SystemStat
        message  Message
        stopped  int32
    }
)

func (a *actorContext) Actor() Actor {
    return a.actor
}

func (a *actorContext) Id() int {
    return a.id
}

func (a *actorContext) Message() Message {
    return a.message
}

func (a *actorContext) dispatch(message Message) error {
    if atomic.LoadInt32(&a.stopped) == 1 {
        return errors.New("message actor is closed")
    }
    sz := len(a.messages)
    if a.config.ActorQueueSize != 0 && sz >= a.config.ActorQueueSize {
        return errors.New("message actor is full")
    }
    a.messages <- message
    return nil
}
func (a *actorContext) fire(message Message) {
    defer func() {
        recover()
        a.wg.Done()
        atomic.AddInt64(&a.stat.Running, -1)
    }()
    a.wg.Add(1)
    atomic.AddInt64(&a.stat.Running, 1)
    atomic.AddUint64(&a.stat.Finished, 1)
    a.message = message
    a.actor.Receive(a)
    a.message = nil
}
func (a *actorContext) pollingMessage(started chan bool) {
    a.wg.Add(1)
    defer a.wg.Done()
    started <- true

    // On Start
    if p, ok := a.actor.(Startable); ok {
        p.Start(a)
    }
    // polling messages
    for message := range a.messages {
        a.fire(message)
    }
    // On Stop
    if p, ok := a.actor.(Stopable); ok {
        p.Stop(a)
    }
}
func (a *actorContext) OnStart() {
    var started = make(chan bool)
    go a.pollingMessage(started)
    <-started
}
func (a *actorContext) OnStop() {
    if atomic.CompareAndSwapInt32(&a.stopped, 0, 1) {
        close(a.messages)
    }
}
func (a *actorContext) StopIfFree() bool {
    sz := len(a.messages)
    if sz > 0 {
        return false
    }
    a.OnStop()
    return true
}

func newActorInstance(id int, stat *SystemStat, wg *sync.WaitGroup, config *Config) *actorContext {
    actor := &actorContext{
        id:       id,
        messages: make(chan Message, config.ActorQueueSize),
        config:   config,
        wg:       wg,
        stat:     stat,
    }
    return actor
}
