package jobhandler
import(
    "context"
    "testing"
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
