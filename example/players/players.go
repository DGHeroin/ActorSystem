package main

import (
    "fmt"
    "github.com/DGHeroin/ActorSystem/actor"
    "log"
)

func main() {
    sys := actor.NewSystem("players_system", &actor.Config{})
    sys.Start()

    aidA, _ := sys.NewActor()
    aidB, _ := sys.NewActor()

    err := sys.Dispatch(&Message{"random hello"})
    if err != nil {
        fmt.Println(err)
    }
    err = sys.DispatchTo(aidA, &Message{text: "hello A"})
    if err != nil {
        fmt.Println(err)
    }
    err = sys.DispatchTo(aidB, &Message{text: "hello B"})
    if err != nil {
        fmt.Println(err)
    }

    sys.Stop()
}

type (
    Message struct {
        text string
    }
)

func (m *Message) Execute(ctx actor.Context) {
    log.Println("say:", m.text, ctx.Actor())
}
