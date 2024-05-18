# JobHandler

A simple package that enables your program to stop job-creation when the jobhandler is stopped.
The primary usecase for this is graceful shutdowns of services that you do not want to kill forcibly.

In essence JobHandler is a wrapper around sync.WaitGroup with added functionality.

## Caveats
Note that this for this package to work as intended, you must take great care always to report jobs as done upon completion. Also your code cannot have goroutines that may sleep in perpetuity take on jobs. Such goroutines should only attempt TryJob() after wake-up or wake up when the context passed to new jobhandler is cancelled.

## Usage

Simple graceful shutdown:
```go
}
```
