package ActorSystem

import (
    "fmt"
    "sync"
    "sync/atomic"
)

type (
    ActorSystem struct {
        name       string
        dispatcher Actor
        config     *Config
        wg         *sync.WaitGroup
        stat       *SystemStat
    }
    Config struct {
        MinActor          int  `env:"min_actor" default:"10"`
        MaxActor          int  `env:"max_actor" default:"100"`
        TaskQueueSize     int  `env:"task_queue_size" default:"10"`
        DispatchQueueSize int  `env:"dispatch_queue_size" default:"100"`
        DispatchBlocking  bool `env:"dispatch_blocking" default:"false"`
    }
    SystemStat struct {
        Running  int64
        Finished uint64
        Failed   uint64
        Actors   int
    }
)

func (s *ActorSystem) Run() {
    go s.dispatcher.Start()
}

func (s *ActorSystem) SubmitTask(task Task) error {
    err := s.dispatcher.AddTask(task)
    if err != nil {
        atomic.AddUint64(&s.stat.Failed, 1)
    }
    return err
}

func (s *ActorSystem) Shutdown() {
    s.dispatcher.Stop()
    s.wg.Wait()
}
func (s *ActorSystem) Stat() SystemStat {
    return *s.stat
}
func (s SystemStat) String() string {
    return fmt.Sprintf("system stat:\n=>finished tasks:%d\n=>submit failed:%d\n=>running tasks:%d\n=>actors:%v",
        s.Finished, s.Failed, s.Running, s.Actors)
}
func NewActorSystem(name string, config *Config) *ActorSystem {
    var (
        wg   = &sync.WaitGroup{}
        stat = &SystemStat{}
    )
    system := &ActorSystem{
        name:       name,
        config:     config,
        dispatcher: NewDispatcherActor(stat, wg, config),
        wg:         wg,
        stat:       stat,
    }
    return system
}
