package main

import (
    "github.com/DGHeroin/ActorSystem/actor"
    "log"
    "math/rand"
    "sync/atomic"
    "time"
)

func init() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
}
func main() {
    attackSys := actor.NewSystem("attack_system", &actor.Config{
        MinActor:          3,
        MaxActor:          10,
        DispatchQueueSize: 100,
        ActorQueueSize:    20,
        SpawnActor: func() actor.Actor {
            return &AttackActor{}
        },
    })

    attackSys.Start()

    id := 0
    runCount := 0
    failCount := 0
    okCount := 0
    publishDuration := time.Millisecond * 10
    isFastMode := true
    runCount++
    if attackSys.Dispatch(&HealingMessage{HP: 50}) != nil {
        failCount++
    } else {
        okCount++
    }
    for {
        id++
        runCount++
        err := attackSys.Dispatch(&AttackMessage{
            id: id,
            HP: 100,
        })
        if err != nil {
            failCount++
            log.Println("dispatch message failed:", err, id, failCount)
            time.Sleep(time.Second)
        } else {
            okCount++
        }
        time.Sleep(publishDuration)
        if !isFastMode && runCount > 3 {
            publishDuration = time.Millisecond * 10
        }
        if runCount > 300 {
            isFastMode = !isFastMode
            if isFastMode {
                publishDuration = time.Millisecond * time.Duration(rand.Intn(20))
            } else {
                publishDuration = time.Second
            }
            runCount = 0
        }
        if id >= 999 {
            break
        }
    }

    log.Println("start shutdown", execCount, id, failCount, attackSys)
    attackSys.Stop()
    log.Println("finished shutdown", execCount, id, failCount, attackSys)
}

type (
    AttackMessage struct {
        id int
        HP int
    }
    HealingMessage struct {
        HP int
    }
    AttackActor struct {
    }
)

func (a *AttackActor) Receive(ctx actor.Context) {
    switch msg := ctx.Message().(type) {
    case *AttackMessage:
        msg.HP -= 50 + rand.Intn(5)
        atomic.AddInt32(&execCount, 1)
        time.Sleep(time.Millisecond * time.Duration(50+rand.Intn(150)))
    case *HealingMessage:
        msg.HP += 30 + rand.Intn(10)
        log.Println("do healing", msg.HP)
    }
}

func (a *AttackActor) Start(ctx actor.Context) {
    log.Println("AttackActor Start:", ctx.Id())
}

func (a *AttackActor) Stop(ctx actor.Context) {
    log.Println("AttackActor Stop:", ctx.Id())
}

var execCount int32
