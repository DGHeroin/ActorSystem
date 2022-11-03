package main

import (
    "github.com/DGHeroin/ActorSystem"
    "log"
    "sync/atomic"
    "time"
)

func init() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
}
func main() {
    attackSys := ActorSystem.NewSystem("attack_system", &ActorSystem.Config{
        MinActor:          3,
        MaxActor:          30,
        DispatchQueueSize: 30,
        ActorQueueSize:    10,
    })
    attackSys.Run()
    id := 0
    runCount := 0
    failCount := 0
    okCount := 0
    publishDuration := time.Millisecond * 10
    isFastMode := true
    for {
        id++
        runCount++
        err := attackSys.Dispatch(&AttackMessage{id: id})
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
                publishDuration = time.Millisecond * 10
            } else {
                publishDuration = time.Second
            }
            runCount = 0
        }
        if id >= 1000 {
            break
        }
    }

    log.Println("start shutdown", execCount, id, failCount, attackSys)
    attackSys.Shutdown()
    log.Println("finished shutdown", execCount, id, failCount, attackSys)
}

type (
    AttackMessage struct {
        id int
    }
)

var execCount int32

func (a *AttackMessage) Execute() {
    // log.Println("do attack:", a.id)
    atomic.AddInt32(&execCount, 1)
    time.Sleep(time.Second)
}
