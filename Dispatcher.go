package ActorSystem

import (
    "errors"
    "sync"
    "time"
)

type Dispatcher struct {
    pool     []Actor
    messages chan Message
    index    int
    mu       sync.Mutex
    config   *Config
    wg       *sync.WaitGroup
    stopChan chan bool
    stat     *SystemStat
}

func (a *Dispatcher) Receive(message Message) error {
    sz := len(a.messages)
    if sz >= a.config.DispatchQueueSize {
        // 检查还有没有额度创建新 message actor
        if !a.growActor() {
            if !a.config.DispatchBlocking {
                return errors.New("message dispatcher queue is full")
            }
        }
    }
    a.messages <- message
    return nil
}
func (a *Dispatcher) Start() {
    for message := range a.messages {
        for {
            var actor = a.Pick() // pick actor form actors pool
            if actor != nil {
                if err := actor.Receive(message); err == nil {
                    break
                }
            }
        }
    }
}
func (a *Dispatcher) Stop() {
    close(a.messages)
    a.mu.Lock()
    defer a.mu.Unlock()
    for _, actor := range a.pool {
        actor.Stop()
    }
    a.stopChan <- true
}
func (a *Dispatcher) Pick() Actor {
    a.mu.Lock()
    defer a.mu.Unlock()
    if len(a.pool) == 0 {
        return nil
    }
    a.index = a.index % len(a.pool)
    actor := a.pool[a.index]
    a.index += 1
    return actor
}
func (a *Dispatcher) autoScale() {
    for {
        select {
        case <-a.stopChan:
            return
        case <-time.After(100 * time.Millisecond):
            if a.actorsSize() < a.config.MinActor {
                a.growActor()
            }
            if a.queueSize() == 0 {
                if a.RemoveActor(1) {
                }
            }
        }
    }
}
func (a *Dispatcher) AddActor(dt int) {
    a.mu.Lock()
    defer a.mu.Unlock()
    for i := 0; i < dt; i++ {
        a.pool = append(a.pool, NewActor(a.stat, a.wg, a.config))
    }
    a.stat.Actors = len(a.pool)
}
func (a *Dispatcher) RemoveActor(dt int) bool {
    if a.actorsSize() <= a.config.MinActor {
        return false
    }

    a.mu.Lock()
    actors := a.pool[:dt]
    a.pool = a.pool[dt:]
    a.mu.Unlock()

    for _, actor := range actors {
        actor.Stop()
    }
    a.stat.Actors = len(a.pool)
    return true
}
func (a *Dispatcher) actorsSize() int {
    a.mu.Lock()
    defer a.mu.Unlock()
    return len(a.pool)
}
func (a *Dispatcher) queueSize() int {
    a.mu.Lock()
    defer a.mu.Unlock()
    return len(a.messages)
}
func (a *Dispatcher) growActor() bool {
    if a.actorsSize() >= a.config.MaxActor {
        return false
    }
    a.AddActor(1)
    return true
}
func NewDispatcherActor(stat *SystemStat, wg *sync.WaitGroup, config *Config) Actor {
    actor := &Dispatcher{
        messages: make(chan Message, config.DispatchQueueSize),
        pool:     make([]Actor, config.MinActor),
        config:   config,
        wg:       wg,
        stopChan: make(chan bool),
        stat:     stat,
    }
    for i := 0; i < config.MinActor; i++ {
        actor.pool[i] = NewActor(stat, wg, config)
    }
    stat.Actors = len(actor.pool)
    go actor.autoScale()
    return actor
}
