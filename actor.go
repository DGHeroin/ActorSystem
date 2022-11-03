package ActorSystem

type (
    Actor interface {
        Receive(ctx Context)
    }
    Message interface{}
    Context interface {
        Id() int
        Message() Message
        Actor() Actor
    }
    Startable interface {
        Start(ctx Context)
    }
    Stopable interface {
        Stop(ctx Context)
    }
    Executable interface {
        Execute(ctx Context)
    }
)

type (
    actorDummy struct{}
)

func (a actorDummy) Receive(ctx Context) {
    switch msg := ctx.Message().(type) {
    case Executable:
        msg.Execute(ctx)
    }
}
func (a actorDummy) Start(Context) {}
func (a actorDummy) Stop(Context)  {}
