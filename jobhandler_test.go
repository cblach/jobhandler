package jobhandler
import(
    "context"
    "slices"
    "sync/atomic"
    "testing"
    "time"
)

func TestZeroHandler(t *testing.T) {
    var jh JobHandler
    if jh.Try() {
        t.Fatal("zero handler should not accept jobs")
    }
    if !jh.Stopped() {
        t.Fatal("zero handler should be stopped")
    }
    jh.WaitAll()
    jh.Stop()
    jh.WaitAll()
}

func TestJob(t *testing.T) {
    t.Run("done=>stop=>wait", func (t *testing.T) {
        jh := New(context.Background())
        if !jh.Try() {
            t.Fatal("unable to try")
        }
        jh.Done()
        jh.Stop()
        jh.WaitAll()
    })
    t.Run("stop=>done=>wait", func (t *testing.T) {
        jh := New(context.Background())
        if !jh.Try() {
            t.Fatal("unable to try")
        }
        jh.Stop()
        jh.Done()
        jh.WaitAll()
    })
    t.Run("wait=>done=>stop", func (t *testing.T) {
        jh := New(context.Background())
        if !jh.Try() {
            t.Fatal("unable to try")
        }
        go func () {
            jh.Done()
            jh.Stop()
        }()
        jh.WaitAll()
    })
    t.Run("wait=>stop=>done", func (t *testing.T) {
        jh := New(context.Background())
        if !jh.Try() {
            t.Fatal("unable to try")
        }
        go func () {
            jh.Stop()
            jh.Done()

        }()
        jh.WaitAll()
    })
}

func TestJobs(t *testing.T) {
    jh := New(context.Background())
    nJobs := 10
    if !jh.TryN(nJobs) {
        t.Fatal("unable to try")
    }
    for i := 0; i < nJobs; i++ {
        jh.Done()
    }
    jh.Stop()
    jh.WaitAll()
}

func TestStop(t *testing.T) {
    jh := New(context.Background())
    if jh.Stopped() {
        t.Fatal("should not be stopped")
    }
    jh.Stop()
    if !jh.Stopped() {
        t.Fatal("should be stopped")
    }
    if jh.Try() {
        t.Fatal("stopped handler should not accept jobs")
    }
    jh.WaitAll()
}

func TestContext(t *testing.T) {
    t.Run("wait=>cancel", func (t *testing.T) {
        ctx, cancel := context.WithCancel(context.Background())
        jh := New(ctx)
        go func () {
            cancel()
        }()
        jh.WaitAll()
    })
    t.Run("cancel=>wait", func (t *testing.T) {
        ctx, cancel := context.WithCancel(context.Background())
        jh := New(ctx)
        ch := make(chan struct{})
        go func () {
            <-ch
            cancel()
        }()
        ch<-struct{}{}
        jh.WaitAll()
    })
}

func TestTryFunc(t *testing.T) {
    t.Run("open jobhandler", func (t *testing.T) {
        didRunFn := false
        jh := New(context.Background())
        if !jh.TryFunc(func () { didRunFn = true }) {
            t.Fatal("unable to try")
        }
        jh.Stop()
        jh.WaitAll()
        if !didRunFn {
            t.Fatal("function did not run")
        }
    })
    t.Run("closed jobhandler", func (t *testing.T) {
        didRunFn := false
        jh := New(context.Background())
        jh.Stop()
        if jh.TryFunc(func () { didRunFn = true }) {
            t.Fatal("should not accept jobs")
        }
        jh.WaitAll()
        if didRunFn {
            t.Fatal("function did run")
        }
    })
}

func TestTryFuncAsync(t *testing.T) {
    t.Run("open jobhandler", func (t *testing.T) {
        didRunFn := false
        jh := New(context.Background())
        if !<-jh.TryFuncAsync(func () { didRunFn = true }) {
            t.Fatal("unable to try")
        }
        jh.Stop()
        jh.WaitAll()
        if !didRunFn {
            t.Fatal("function did not run")
        }
    })
    t.Run("closed jobhandler", func (t *testing.T) {
        didRunFn := false
        jh := New(context.Background())
        jh.Stop()
        if <-jh.TryFuncAsync(func () { didRunFn = true }) {
            t.Fatal("should not accept jobs")
        }
        jh.WaitAll()
        if didRunFn {
            t.Fatal("function did run")
        }
    })
}

func TestTryNFuncAsync(t *testing.T) {
    t.Run("open jobhandler", func (t *testing.T) {
        didRunFn := false
        jh := New(context.Background())
        if !<-jh.TryNFuncAsync(1, 1, func (i int) { didRunFn = true }) {
            t.Fatal("unable to try")
        }
        jh.Stop()
        jh.WaitAll()
        if !didRunFn {
            t.Fatal("function did not run")
        }
    })
    t.Run("closed jobhandler", func (t *testing.T) {
        didRunFn := false
        jh := New(context.Background())
        jh.Stop()
        if <-jh.TryNFuncAsync(1, 1, func (i int) { didRunFn = true }) {
            t.Fatal("should not accept jobs")
        }
        jh.WaitAll()
        if didRunFn {
            t.Fatal("function did run")
        }
    })
    t.Run("open jobhandler: limit", func (t *testing.T) {
        delta := 11
        limit := 5
        ch := make(chan struct{})
        arr := make([]int, 0, delta)
        var nRunning atomic.Int32
        jh := New(context.Background())
        if !<-jh.TryNFuncAsync(delta, limit, func (i int) {
            nRunning.Add(1)
            ch <- struct{}{}
            arr = append(arr, i)
            nRunning.Add(-1)
        }) {
            t.Fatal("unable to try")
        }
        go func () {
            for len(arr) < delta {
                for int(nRunning.Load()) < min(limit, delta - len(arr)) {
                    time.Sleep(1 * time.Millisecond)
                }
                time.Sleep(10 * time.Millisecond)
                if int(nRunning.Load()) != min(limit, delta - len(arr)) {
                    panic("unexpected running count")
                }
                <-ch
            }
        }()
        jh.Stop()
        jh.WaitAll()
        slices.Sort(arr)
        if len(arr) != delta {
            t.Fatal("unexpected length", len(arr))
        }
        for i := 0; i < delta; i++ {
            if arr[i] != i {
                t.Fatal("unexpected value", arr[i])
            }
        }
    })
}

func TestNegativeJobs(t *testing.T) {
    jh := New(context.Background())
    if !jh.TryN(0) {
        t.Fatal("should accept zero job")
    }
    if jh.TryN(-1) {
        t.Fatal("should not accept negative jobs")
    }
    jh.Stop()
    jh.WaitAll()
}

func TestJobPanic(t *testing.T) {
    t.Run("0 jobs", func (t *testing.T) {
        jh := New(context.Background())
        jh.TryN(1)
        jh.Done()
        defer func() {
            if r := recover(); r == nil {
                t.Fatal("should panic")
            } else if r.(string) != "zero job count while running, should be at least 1" {
                t.Fatal("unexpected panic", r)
            }
        }()
        jh.Done()
    })
    t.Run("done: -1 jobs", func (t *testing.T) {
        var jh JobHandler
        defer func() {
            if r := recover(); r == nil {
                t.Fatal("should panic")
            } else if r.(string) != "negative job count" {
                t.Fatal("unexpected panic", r)
            }
        }()
        jh.Done()
    })
}
