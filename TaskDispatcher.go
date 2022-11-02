package ActorSystem

import (
    "errors"
    "sync"
    "time"
)

type TaskDispatcher struct {
    pool     []Actor
    tasks    chan Task
    index    int
    mu       sync.Mutex
    config   *Config
    wg       *sync.WaitGroup
    stopChan chan bool
    stat     *SystemStat
}

func (a *TaskDispatcher) AddTask(task Task) error {
    sz := len(a.tasks)
    if sz >= a.config.DispatchQueueSize {
        // 检查还有没有额度创建新 task actor
        if !a.growTaskActor() {
            if !a.config.DispatchBlocking {
                return errors.New("task dispatcher queue is full")
            }
        }
    }

    a.tasks <- task
    return nil
}
func (a *TaskDispatcher) Start() {
    for task := range a.tasks {
        for {
            var actor = a.Pick() // pick actor form actors pool
            if actor != nil {
                err := actor.AddTask(task)
                if err == nil {
                    break
                }
            }
        }
    }
}
func (a *TaskDispatcher) Stop() {
    close(a.tasks)
    a.mu.Lock()
    defer a.mu.Unlock()
    for _, actor := range a.pool {
        actor.Stop()
    }
    a.stopChan <- true
}
func (a *TaskDispatcher) Pick() Actor {
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

func (a *TaskDispatcher) autoScale() {
    for {
        select {
        case <-a.stopChan:
            return
        case <-time.After(100 * time.Millisecond):
            if a.actorsSize() < a.config.MinActor {
                a.growTaskActor()
            }
            if a.queueSize() == 0 {
                if a.RemoveActor(1) {
                }
            }
        }
    }
}
func (a *TaskDispatcher) AddActor(dt int) {
    a.mu.Lock()
    defer a.mu.Unlock()
    for i := 0; i < dt; i++ {
        a.pool = append(a.pool, NewTaskActor(a.stat, a.wg, a.config))
    }
    a.stat.Actors = len(a.pool)
}
func (a *TaskDispatcher) RemoveActor(dt int) bool {
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
func (a *TaskDispatcher) actorsSize() int {
    a.mu.Lock()
    defer a.mu.Unlock()
    return len(a.pool)
}
func (a *TaskDispatcher) queueSize() int {
    a.mu.Lock()
    defer a.mu.Unlock()
    return len(a.tasks)
}
func (a *TaskDispatcher) growTaskActor() bool {
    if a.actorsSize() >= a.config.MaxActor {
        return false
    }
    a.AddActor(1)
    return true
}
func NewDispatcherActor(stat *SystemStat, wg *sync.WaitGroup, config *Config) Actor {
    actor := &TaskDispatcher{
        tasks:    make(chan Task, config.DispatchQueueSize),
        pool:     make([]Actor, config.MinActor),
        config:   config,
        wg:       wg,
        stopChan: make(chan bool),
        stat:     stat,
    }
    for i := 0; i < config.MinActor; i++ {
        actor.pool[i] = NewTaskActor(stat, wg, config)
    }
    stat.Actors = len(actor.pool)
    go actor.autoScale()
    return actor
}
