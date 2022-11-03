package ActorSystem

import (
    "fmt"
    "sync"
    "sync/atomic"
)

type (
    MessageSystem struct {
        name       string
        dispatcher Actor
        config     *Config
        wg         *sync.WaitGroup
        stat       *SystemStat
    }
    Config struct {
        MinActor          int
        MaxActor          int
        ActorQueueSize    int
        DispatchQueueSize int
        DispatchBlocking  bool
    }
    SystemStat struct {
        Running  int64
        Finished uint64
        Failed   uint64
        Actors   int
    }
)

func (s *MessageSystem) Run() {
    go s.dispatcher.Start()
}

func (s *MessageSystem) Dispatch(message Message) error {
    err := s.dispatcher.Receive(message)
    if err != nil {
        atomic.AddUint64(&s.stat.Failed, 1)
    }
    return err
}

func (s *MessageSystem) Shutdown() {
    s.dispatcher.Stop()
    s.wg.Wait()
}
func (s MessageSystem) String() string {
    return fmt.Sprintf("[%s] system stat:\n%s", s.name, s.stat.String())
}
func (s *MessageSystem) Stat() SystemStat {
    return *s.stat
}
func (s SystemStat) String() string {
    return fmt.Sprintf("=>finished messages:%d\n=>submit failed:%d\n=>running messages:%d\n=>actors:%v",
        s.Finished, s.Failed, s.Running, s.Actors)
}
func NewSystem(name string, config *Config) System {
    var (
        wg   = &sync.WaitGroup{}
        stat = &SystemStat{}
    )
    system := &MessageSystem{
        name:       name,
        config:     config,
        dispatcher: NewDispatcherActor(stat, wg, config),
        wg:         wg,
        stat:       stat,
    }
    return system
}
