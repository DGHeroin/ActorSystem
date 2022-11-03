package main

import (
    "fmt"
    "github.com/DGHeroin/ActorSystem"
    "log"
)

func main() {
    sys := ActorSystem.NewSystem("players_system", &ActorSystem.Config{})
    sys.Start()

    A, _ := sys.NewActor()
    B, _ := sys.NewActor()

    err := sys.Dispatch(&Message{"random hello"})
    if err != nil {
        fmt.Println(err)
    }
    err = sys.DispatchTo(A, &Message{text: "hello A"})
    if err != nil {
        fmt.Println(err)
    }
    err = sys.DispatchTo(B, &Message{text: "hello B"})
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

func (m *Message) Execute(ctx ActorSystem.Context) {
    log.Println("say:", m.text, ctx.Actor())
}
