package barbarian

import (
	"sync/atomic"
	"time"
)

// Statistics records runtime data for the Barbarian.
type Statistics struct {
	Start         time.Time
	Finish        time.Time
	UpTime        time.Duration
	Inputs        int
	Execps        float64
	Execs         uint64
	PositiveExecs uint64
}

// RunHandler defines the function signature for the main work handler.
type RunHandler func(line string) interface{}

// ResultHandler defines the function signature for the result processing handler.
type ResultHandler func(result interface{})

// Barbarian manages and executes brute-force attacks or similar tasks concurrently.
type Barbarian struct {
	Stats      Statistics
	runHandler RunHandler
	resHandler ResultHandler
	workList   []string
	blockers   chan struct{}
	input      chan string
	output     chan interface{}
	stop       chan struct{}
}

// New creates and returns a new Barbarian instance.
func New(runHandler RunHandler, resHandler ResultHandler, workList []string, concurrency int) *Barbarian {
	return &Barbarian{
		Stats:      Statistics{},
		runHandler: runHandler,
		resHandler: resHandler,
		workList:   workList,
		blockers:   make(chan struct{}, concurrency),
		input:      make(chan string),
		output:     make(chan interface{}),
		stop:       make(chan struct{}),
	}
}

// Run starts the Barbarian brute-force operation.
func (bb *Barbarian) Run() {
	bb.Stats.Start = time.Now()

	// Handle input processing concurrently.
	go func() {
		for in := range bb.input {
			go func(in string) {
				atomic.AddUint64(&bb.Stats.Execs, 1)
				res := bb.runHandler(in)
				if res != nil {
					bb.output <- res
					atomic.AddUint64(&bb.Stats.PositiveExecs, 1)
				}
				<-bb.blockers
			}(in)
		}
	}()

	// Handle output processing concurrently.
	go func() {
		for res := range bb.output {
			bb.resHandler(res)
		}
	}()

	bb.Stats.Inputs = len(bb.workList)
	for _, line := range bb.workList {
		select {
		case <-bb.stop:
			close(bb.input)
			return
		default:
			bb.blockers <- struct{}{}
			bb.input <- line
		}
	}

	// Wait for all goroutines to finish.
	for i := 0; i < cap(bb.blockers); i++ {
		bb.blockers <- struct{}{}
	}
	bb.Report()
}

// Stop gracefully stops the Barbarian operation.
func (bb *Barbarian) Stop() {
	close(bb.stop)
}

// Report finalizes and reports the runtime statistics.
func (bb *Barbarian) Report() {
	bb.Stats.Finish = time.Now()
	bb.Stats.UpTime = bb.Stats.Finish.Sub(bb.Stats.Start)
	bb.Stats.Execps = float64(bb.Stats.Execs) / bb.Stats.UpTime.Seconds()
}
