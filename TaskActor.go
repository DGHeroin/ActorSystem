package ActorSystem

import (
    "errors"
    "sync"
    "sync/atomic"
)

type TaskActor struct {
    id        int32
    tasks     chan Task
    config    *Config
    closeFlag int32
    wg        *sync.WaitGroup
    stat      *SystemStat
}

func (a *TaskActor) AddTask(task Task) error {
    sz := len(a.tasks)
    if sz >= a.config.TaskQueueSize {
        return errors.New("task actor is full")
    }
    if atomic.LoadInt32(&a.closeFlag) == 2 {
        return errors.New("task actor is closed")
    }
    a.tasks <- task
    return nil
}
func (a *TaskActor) fire(task Task) {
    defer func() {
        recover()
        a.wg.Done()
        atomic.AddInt64(&a.stat.Running, -1)
    }()
    a.wg.Add(1)
    atomic.AddInt64(&a.stat.Running, 1)
    atomic.AddUint64(&a.stat.Finished, 1)

    task.Execute()
}
func (a *TaskActor) Start() {
    for task := range a.tasks {
        a.fire(task)
    }
}
func (a *TaskActor) Stop() {
    if atomic.CompareAndSwapInt32(&a.closeFlag, 1, 2) {
        close(a.tasks)
    }
}

func (a *TaskActor) StopIfFree() bool {
    sz := len(a.tasks)
    if sz > 0 {
        return false
    }
    a.Stop()
    return true
}

var taskActorId int32

func NewTaskActor(stat *SystemStat, wg *sync.WaitGroup, config *Config) Actor {
    actor := &TaskActor{
        id:     atomic.AddInt32(&taskActorId, 1),
        tasks:  make(chan Task, config.TaskQueueSize),
        config: config,
        wg:     wg,
        stat:   stat,
    }
    go actor.Start()
    return actor
}
