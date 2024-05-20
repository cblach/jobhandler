package jobhandler_test
import(
    "context"
    "fmt"
    "github.com/cblach/jobhandler"
    "slices"
    "sync/atomic"
)

func ExampleJobHandler_Try() {
    jh := jobhandler.New(context.Background())
    if !jh.Try() {
        return
    }
    fmt.Println("did some job...")
    jh.Done()
    jh.Stop()
    jh.WaitAll()
    // Output: did some job...
}

func ExampleJobHandler_TryN() {
    jh := jobhandler.New(context.Background())
    if !jh.TryN(10) {
        return
    }
    arr := make([]int, 10)
    go func() {
        for i := 0; i < 10; i++ {
            arr[i] = i
            jh.Done()
        }
    }()
    jh.Stop()
    jh.WaitAll()
    slices.Sort(arr)
    for i := 0; i < 10; i++ {
        fmt.Print(arr[i])
    }
    // Output: 0123456789
}

func ExampleJobHandler_TryFunc() {
    jh := jobhandler.New(context.Background())
    if !jh.TryFunc(func () {
        fmt.Println("did some job...")
    }) {
        fmt.Println("failed to take on job")
    }
    jh.Stop()
    jh.WaitAll()
    // Output: did some job...
}

func ExampleJobHandler_TryFuncAsync() {
    jh := jobhandler.New(context.Background())
    if !<-jh.TryFuncAsync(func () {
        fmt.Println("did some job...")
    }) {
        fmt.Println("failed to take on job")
    }
    jh.Stop()
    jh.WaitAll()
    // Output: did some job...
}

func ExampleJobHandler_TryNFuncAsync() {
    jh := jobhandler.New(context.Background())
    var sum atomic.Uint64
    isPrime := func(n int) bool {
        if n <= 1 {
            return false
        }
        for i := 2; i < n; i++ {
            if n % i == 0 {
                return false
            }
        }
        return true
    }
    // sum all primes from 1 to 100
    if !<-jh.TryNFuncAsync(100, 5, func (i int) {
        if isPrime(i) {
            sum.Add(uint64(i))
        }
    }) {
        fmt.Println("failed to take on job")
    }
    jh.Stop()
    jh.WaitAll()
    fmt.Print("prime sum for 1..100 is ", sum.Load())
    // Output: prime sum for 1..100 is 1060
}
