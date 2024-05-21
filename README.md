# JobHandler
![gol½ang tests status](https://github.com/cblach/jobhandler/actions/workflows/go.yml/badge.svg)

A simple package that enables your program to stop job-creation when the jobhandler is stopped.
The primary usecase for this is graceful shutdowns of services that you do not want to kill forcibly.

In essence JobHandler is a wrapper around sync.WaitGroup with added functionality.

## Caveats
Note that this for this package to work as intended, you must take great care always to report jobs as done upon completion. Also your code cannot have goroutines that may sleep in perpetuity after taking on jobs. Such goroutines should only attempt TryJob() after wake-up or wake up when the context passed to new jobhandler is cancelled.

## Usage

Basic usage with context:
```go
package main
import(
    "context"
    "fmt"
    "github.com/cblach/jobhandler"
    "time"
)

var JobHandler *jobhandler.JobHandler

func maybeRunJob() {
    if !JobHandler.Try() {
        return
    }
    go func () {
        defer JobHandler.Done()
        fmt.Println("running critical job...")
        time.Sleep(5 * time.Second)
        fmt.Println("... and done")
    }()
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    JobHandler = jobhandler.New(ctx)
    maybeRunJob()
    cancel()
    JobHandler.WaitAll()
}
```

Simple graceful shutdown upon receiving SIGTERM:
```go
package main
import(
    "context"
    "fmt"
    "github.com/cblach/jobhandler"
    "os"
    "os/signal"
    "syscall"
    "time"
)
func main() {
    ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
    jh := jobhandler.New(ctx)
    go func () {
        for {
            if !jh.TryFunc(func () {
                fmt.Println("running critical job...")
                time.Sleep(5 * time.Second)
                fmt.Println("... and done")
            }) {
                break
            }
        }
    }()
    jh.WaitAll()
    os.Exit(0)
}
```
