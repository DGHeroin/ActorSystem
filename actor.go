package ActorSystem

import "sync"

type (
    Actor interface {
        AddTask(task Task) error
        Start()
        Stop()
    }
    System interface {
        Run()
        SubmitTas(task Task)
        Shutdown(shutdownWG *sync.WaitGroup)
    }
    Task interface {
        Execute()
    }
)
