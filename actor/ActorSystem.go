package actor

import (
    "errors"
    "fmt"
    "sync"
    "sync/atomic"
    "time"
)

type (
    System struct {
        name          string
        config        *Config
        wg            *sync.WaitGroup
        actorId       int
        pool          []*actorContext
        m             map[int]*actorContext
        messages      chan *messageItem
        index         int
        mu            sync.Mutex
        stopChan      chan bool
        stat          *SystemStat
        stopped       int32
        stopPollingCh chan bool
    }
    Config struct {
        MinActor          int
        MaxActor          int
        ActorQueueSize    int
        DispatchQueueSize int
        DispatchBlocking  bool
        SpawnActor        func() Actor
    }
    SystemStat struct {
        Running  int64
        Finished uint64
        Failed   uint64
        Actors   int
    }
    messageItem struct {
        actorId int
        message Message
    }
)

func (sys *System) Start() {
    go sys.autoJustActorPool()
    ch := make(chan bool)
    go sys.pollingMessage(ch)
    <-ch
}

func (sys *System) String() string {
    return fmt.Sprintf("[%s] system stat:\n%s", sys.name, sys.stat.String())
}
func (sys *System) Stat() SystemStat {
    return *sys.stat
}
func (a SystemStat) String() string {
    return fmt.Sprintf(`
=>total:%d
=>finished messages:%d
=>submit failed:%d
=>running messages:%d
=>actors:%v`,
        a.Finished+a.Failed,
        a.Finished, a.Failed, a.Running, a.Actors)
}
func (sys *System) Dispatch(message Message) error {
    sz := len(sys.messages)
    if sys.config.DispatchQueueSize != 0 && sz >= sys.config.DispatchQueueSize {
        // 检查还有没有额度创建新 message actor
        if !sys.addActor(1) {
            if !sys.config.DispatchBlocking {
                atomic.AddUint64(&sys.stat.Failed, 1)
                return errors.New("message dispatcher queue is full")
            }
        }
    }
    sys.messages <- &messageItem{
        message: message,
    }
    return nil
}
func (sys *System) DispatchTo(id int, message Message) error {
    if _, ok := sys.pickActorWithId(id); !ok {
        return errors.New("actor not exist")
    }

    sys.messages <- &messageItem{
        actorId: id,
        message: message,
    }
    return nil
}
func (sys *System) pollingMessage(ch chan bool) {
    ch <- true
    for it := range sys.messages {
        for { // pick actor form actors pool
            if actor := sys.pickActorRoundRobin(); actor != nil {
                if err := actor.dispatch(it.message); err == nil {
                    break
                }
            }
        }
    }
    sys.stopPollingCh <- true
}
func (sys *System) Stop() {
    if atomic.CompareAndSwapInt32(&sys.stopped, 0, 1) {
        // stop accept new message
        close(sys.messages)
        // make sure all messages has been dispatch to actor
        // now some messages may not be processed
        <-sys.stopPollingCh
        // stop actors auto-just
        sys.stopChan <- true
        // gently wait all actor processed all messages
        sys.wg.Wait()
    }
}
func (sys *System) pickActorRoundRobin() *actorContext {
    sys.mu.Lock()
    defer sys.mu.Unlock()
    if len(sys.pool) == 0 {
        return nil
    }
    sys.index = sys.index % len(sys.pool)
    actor := sys.pool[sys.index]
    sys.index += 1
    return actor
}
func (sys *System) pickActorWithId(id int) (*actorContext, bool) {
    sys.mu.Lock()
    defer sys.mu.Unlock()
    if it, ok := sys.m[id]; ok {
        return it, true
    }
    return nil, false
}
func (sys *System) freeAllActor() {
    sys.mu.Lock()
    defer sys.mu.Unlock()
    for _, actor := range sys.pool {
        actor.OnStop()
    }
}
func (sys *System) autoJustActorPool() {
out:
    for {
        select {
        case <-sys.stopChan:
            break out
        case <-time.After(100 * time.Millisecond):
            if sys.config.MinActor == 0 && sys.config.MaxActor == 0 {
                continue
            }
            if sys.actorsSize() < sys.config.MinActor {
                sys.addActor(1)
            }
            if sys.queueSize() == 0 {
                sys.removeActor(1)
            }
        }
    }
    sys.freeAllActor()
}
func (sys *System) addActor(dt int) bool {
    if sys.actorsSize() >= sys.config.MaxActor {
        return false
    }
    for i := 0; i < dt; i++ {
        sys.newActor()
    }
    sys.stat.Actors = len(sys.pool)
    return false
}
func (sys *System) removeActor(dt int) bool {
    if sys.actorsSize() <= sys.config.MinActor {
        return false
    }

    sys.mu.Lock()
    actors := sys.pool[:dt]
    sys.pool = sys.pool[dt:]
    for _, actor := range actors {
        delete(sys.m, actor.id)
    }
    sys.mu.Unlock()

    for _, actor := range actors {
        actor.OnStop()
    }

    sys.stat.Actors = len(sys.pool)
    return true
}
func (sys *System) actorsSize() int {
    sys.mu.Lock()
    defer sys.mu.Unlock()
    return len(sys.pool)
}
func (sys *System) queueSize() int {
    sys.mu.Lock()
    defer sys.mu.Unlock()
    return len(sys.messages)
}
func (sys *System) newActor() *actorContext {
    sys.mu.Lock()
    defer sys.mu.Unlock()
    id := sys.actorId
    for {
        if _, ok := sys.m[id]; !ok {
            break
        }
        sys.actorId++
        id = sys.actorId
    }

    var instance = newActorInstance(id, sys.stat, sys.wg, sys.config)
    if sys.config.SpawnActor != nil {
        instance.actor = sys.config.SpawnActor()
    } else {
        instance.actor = &actorDummy{}
    }
    sys.pool = append(sys.pool, instance)
    sys.m[id] = instance
    instance.OnStart()
    return instance
}
func (sys *System) NewActor() (int, Actor) {
    instance := sys.newActor()
    return instance.id, instance.actor
}
func NewSystem(name string, config *Config) *System {
    var (
        wg   = &sync.WaitGroup{}
        stat = &SystemStat{}
    )
    system := &System{
        name:          name,
        config:        config,
        messages:      make(chan *messageItem, config.DispatchQueueSize),
        stopChan:      make(chan bool),
        stopPollingCh: make(chan bool),
        m:             map[int]*actorContext{},
        wg:            wg,
        stat:          stat,
    }
    return system
}
