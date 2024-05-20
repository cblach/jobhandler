package jobhandler

import(
    "context"
    "sync"
    "sync/atomic"
    "time"
)

// A jobhandler accepts new jobs until the jobhandler is stopped, at which point
// any new job is rejected.
// A zero jobhandler is valid, but is considered stopped and will not accept any jobs.
type JobHandler struct {
    n        int64
    stopChan chan struct{}
    running  atomic.Bool
    wg       sync.WaitGroup
}

// Create a new job handler
// The jobhandler is stopped when the passed context is done.
func New(ctx context.Context) *JobHandler {
    jh := JobHandler{
        n:        1,
        stopChan: make(chan struct{}),
    }
    jh.running.Store(true)
    jh.wg.Add(1)
    if ctx != nil && ctx.Done() != nil {
        go func() {
            select {
            case <-ctx.Done():
                jh.Stop()
            case <-jh.stopChan:
            }
        }()
    }
    return &jh
}

// Attempt to take on a single job.
// Returns true if job is successfully taken
// and false if the JobHandler is stopped.
// When the job is done call the Done() method.
func (jh *JobHandler) Try() bool {
    return jh.TryN(1)
}

// TryFunc is a convenience function that combines Try() and Done().
// Returns true if job is successfully taken
// and false if the JobHandler is stopped.
// Do not call Done(), the job is automatically
// flagged as done after fn exits.
func (jh *JobHandler) TryFunc(fn func()) bool {
    if !jh.Try() {
        return false
    }
    fn()
    jh.Done()
    return true
}
// TryN attempts to take on multiple jobs.
// Either all jobs are taken or none are taken.
// Returns true if the jobs are successfully taken
// and false if the JobHandler is stopped.
// Done must be called for each of the delta jobs taken.
func (jh *JobHandler) TryN(delta int) bool {
    if delta < 0 {
        return false
    }
    if !jh.running.Load() {
        return false
    }
    for {
        prev := atomic.LoadInt64(&jh.n)
        if prev < 0 {
            panic("negative job count")
        }
        if prev == 0 {
            return false
        }
        if atomic.CompareAndSwapInt64(&jh.n, prev, prev + int64(delta)) {
            break
        }
    }
    jh.wg.Add(delta)
    return true
}

// TryFuncAsync is a convenience function that
// combines Try() and Done() and runs the function asynchronously.
// Returns a read-only channel that sends a boolean value.
// If the job is successfully taken, the channel sends true.
// If the jobhandler is stopped, the channel sends false.
// Do not call Done(), the job is automatically
// flagged as done after fn exits.
func (jh *JobHandler) TryFuncAsync(fn func()) <-chan bool {
    ch := make(chan bool, 1)
    if !jh.Try() {
        ch <- false
        return ch
    }
    go func() {
        fn()
        jh.Done()
        ch <- true
    }()
    return ch
}

// TryFuncAsync is a convenience function to run coupled jobs.
// It combines TryN() and Done().
// Either all jobs are taken or none are taken.
// Func fn is called delta times with the job index,
// 0 in the first call delta-1 in the final call.
// A goroutine is spawned for each call, but no more than
// limit at a time. If limit is <= 0, it is set to delta.
// The fn is not guaranteed to be called in order.
// If the job is successfully taken, the channel sends true.
// If the jobhandler is stopped, the channel sends false.
// Do not call Done(), the jobs are automatically
// flagged as done after the fns exit.
func (jh *JobHandler) TryNFuncAsync(delta, limit int, fn func (int)) <-chan bool {
    ch := make(chan bool, 1)
    if !jh.TryN(delta) {
        ch <- false
        return ch
    }
    // Note: Replace channels with semaphores
    // when they released into standard library
    if limit <= 0 {limit = delta }
    var limitCh chan struct{}
    if limit < delta {
        limitCh = make(chan struct{}, limit)
        for i := 0; i < limit; i++ {
            limitCh <- struct{}{}
        }
    }
    go func() {
        for i := 0; i < delta; i++ {
            if limit < delta { <-limitCh }
            go func() {
                fn(i)
                jh.Done()
                if limit < delta { limitCh <- struct{}{} }
            }()
        }
    }()
    ch <- true
    return ch
}

// TrySleep attempts to sleep duration d. The sleep is cancelled
// if the jobhandler is stopped. Returns true if sleep was
// done. Returns false if jobhandler was stopped before
// the sleep was done.
func (jh *JobHandler) TrySleep(d time.Duration) bool {
    select {
    case <-jh.stopChan:
        return false
    case <-time.After(d):
        return true
    }
}

// Done must be called when a single job is done, regardless of success..
// Note that Done must not be called when using TryFunc, TryFuncAsync
// and TryNFuncAsync. as the job is automatically flagged as done for these functions.
func (jh *JobHandler) Done() {
    if n := atomic.AddInt64(&jh.n, -1); n < 0 {
        panic("negative job count")
    } else if n == 0 && jh.running.Load() {
        panic("zero job count while running, should be at least 1")
    }
    jh.wg.Add(-1)
}

// WaitAll blocks until all jobs are done and the jobhandler is stopped.
// WaitAll is typically used to wait for a graceful shutdowns, and is
// in that case either in the main function or followed by os.Exit(0).
func (jh *JobHandler) WaitAll() {
    jh.wg.Wait()
}

// Stop a jobhandler.
// Returns true if stop is initiated. Returns false if already stopped.
func (jh *JobHandler) Stop() bool {
    if !jh.running.CompareAndSwap(true, false) {
        return false
    }
    if atomic.AddInt64(&jh.n, -1) < 0 {
        panic("negative job count")
    }
    close(jh.stopChan)
    jh.wg.Add(-1)
    return true
}

// IsStopd returns true if jobhandler is stopped and false if not.
func (jh *JobHandler) Stopped() bool {
    return !jh.running.Load()
}

// OnStop returns a channel that's closed when jobhandler is stopped.
func (jh *JobHandler) OnStop() <-chan struct{} {
    return jh.stopChan
}
