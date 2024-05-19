package jobhandler

import(
    "context"
    "errors"
    "sync"
    "sync/atomic"
    "time"
)

var JobHandlerStoppedError = errors.New("jobhandler stopped")

// A jobhandler enables taking on new jobs until the jobhandler is stopped.
type JobHandler struct {
    ctx      context.Context
    n        int64
    stopChan chan struct{}
    running  atomic.Bool
    wg       sync.WaitGroup
}

// Create a new job handler
// The jobhandler is stopped upon calling when the passed context is done.
// A zero group is valid, but is stopped and will not accept any jobs.
func New(ctx context.Context) *JobHandler {
    jc := JobHandler{
        ctx:      ctx,
        n:        1,
        stopChan: make(chan struct{}),
    }
    jc.running.Store(true)
    jc.wg.Add(1)
    return &jc
}

// Attempt to take on a single job.
// Returns true if job is successfully taken
// and false if the JobHandler is stopped.
// When the job is done call the Done() method.
func (jc *JobHandler) Try() bool {
    return jc.TryN(1)
}

// Attempt to take on a single asynchronous job.
// Returns true if job is successfully taken
// and false if the JobHandler is stopped.
// Do not call Done(), the job is automatically
// flagged as done after fn exits.
func (jc *JobHandler) TryFunc(fn func ()) bool {
    if !jc.Try() {
        return false
    }
    go func() {
        fn()
        jc.Done()
    }()
    return true
}

// TryN attempts to take on multiple asynchronous jobs.
// Returns true if the jobs are successfully taken
// and false if the JobHandler is stopped.
// Done must be called for each of the n jobs taken.
func (jc *JobHandler) TryN(delta int) bool {
    if delta < 0 {
        return false
    }
    if !jc.running.Load() {
        return false
    }
    for {
        prev := atomic.LoadInt64(&jc.n)
        if prev < 0 {
            panic("negative job count")
        }
        if prev == 0 {
            return false
        }
        if atomic.CompareAndSwapInt64(&jc.n, prev, prev + int64(delta)) {
            break
        }
    }
    jc.wg.Add(delta)
    return true
}

// TryNFunc attempts to take on multiple asynchronous jobs.
// Returns true if the jobs are successfully taken
// and false if the JobHandler is stopped. The func fn is called n times
// with the job index, 0 in the first call n-1 in the final call.
// Do not call Done(), the job is automatically
// flagged as done after fn exits..
func (jc *JobHandler) TryNFunc(delta int, fn func (int)) bool {
    if !jc.TryN(delta) {
        return false
    }
    for i := 0; i < delta; i++ {
        go func() {
            fn(i)
            jc.Done()
        }()
    }
    return true
}

// Attempt to sleep duration d. Returns true if sleep was
// done. Returns false if jobhandler was stopped before
// that could happen.
func (jc *JobHandler) TrySleep(d time.Duration) bool {
    select {
    case <-jc.stopChan:
        return false
    case <-time.After(d):
        return true
    }
}

// Call when a single job is done, regardless of success.
func (jc *JobHandler) Done() {
    if n := atomic.AddInt64(&jc.n, -1); n < 0 {
        panic("negative job count")
    } else if n == 0 && jc.running.Load() {
        panic("zero job count while running, should be at least 1")
    }
    jc.wg.Add(-1)
}

// WaitAll blocks until all jobs are done and the jobhandler is stopped.
// WaitAll is typically used to wait for a graceful shutdowns, and is
// in that case typically followed by os.Exit(0)
func (jc *JobHandler) WaitAll() {
    if jc.ctx != nil && jc.ctx.Done() != nil {
        go func() {
            select {
            case <-jc.ctx.Done():
                jc.Stop()
            case <-jc.stopChan:
            }
        }()
    }
    jc.wg.Wait()
}

// Stop a jobhandler.
// Returns true if stop is initiated. Returns false if already stopped.
func (jc *JobHandler) Stop() bool {
    if !jc.running.CompareAndSwap(true, false) {
        return false
    }
    if atomic.AddInt64(&jc.n, -1) < 0 {
        panic("negative job count")
    }
    close(jc.stopChan)
    jc.wg.Add(-1)
    return true
}

// IsStopd returns true if jobhandler is stopped and false if not.
func (jc *JobHandler) Stopped() bool {
    return !jc.running.Load()
}

// OnStop returns a channel that's stopped when jobhandler is stopped
func (jh *JobHandler) OnStop() <-chan struct{} {
    return jh.stopChan
}
