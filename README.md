# JobHandler

A simple package that enables your program to stop job-creation when the jobhandler is stopped.
The primary usecase for this is graceful shutdowns of services that you do not want to kill forcibly.

In essence JobHandler is a wrapper around sync.WaitGroup with added functionality.

## Caveats
Note that this for this package to work as intended, you must take great care always to report jobs as done upon completion. Also your code cannot have goroutines that may sleep in perpetuity take on jobs. Such goroutines should only attempt TryJob() after wake-up or wake up when the context passed to new jobhandler is cancelled.

## Usage

Simple graceful shutdown:
```go
import(
	"math/rand"
	"os/signal"
)
func main() {
	jh := jobhandler.NewJobHandler()

	/* Handle termination */
    var termChan = make(chan os.Signal, 1)
    signal.Notify(termChan, syscall.SIGTERM)
        jobHandler.Close()
    }()

    /* Create jobs */
	mJobs := 10
	if !jh.TryJobs(nJobs) {
		return
	}
	for i := 0; i < mJobs; i++ {
		go func() {
			time.Sleep(rand.Int63(5) * time.Second)
			jh.DoneJob()
		}
	}
	jh.WaitAll()

}
```
