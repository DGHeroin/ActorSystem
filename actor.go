package ActorSystem

type (
    System interface {
        Run()
        Dispatch(message Message) error
        Shutdown()
    }
    Actor interface {
        Receive(Message) error
        Start()
        Stop()
    }
    Message interface {
        Execute()
    }
)
