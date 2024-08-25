package barbarian

import (
	"sync/atomic"
	"time"
)

// Statistics records runtime data.
type Statistics struct {
	Start         time.Time
	Finish        time.Time
	UpTime        time.Duration
	Inputs        int
	Execps        float64
	Execs         uint64
	PositiveExecs uint64
}

// RunHandler handles the main work for each input.
type RunHandler func(line string) interface{}

// ResultHandler handles the processing of results.
type ResultHandler func(result interface{})

// Barbarian is a brute-force machine that processes tasks concurrently.
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

// New returns a new Barbarian instance.
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

// Run starts the Barbarian process, handling tasks concurrently.
func (bb *Barbarian) Run() {
	bb.Stats.Start = time.Now()

	// Start a goroutine to handle input processing.
	go func() {
		for in := range bb.input {
			in := in // Capture the variable to avoid issues in goroutines
			go func() {
				atomic.AddUint64(&bb.Stats.Execs, 1)
				res := bb.runHandler(in)
				if res != nil {
					bb.output <- res
					atomic.AddUint64(&bb.Stats.PositiveExecs, 1)
				}
				<-bb.blockers
			}()
		}
	}()

	// Start a goroutine to handle result processing.
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

	// Wait for all goroutines to finish by filling the blockers channel.
	for i := 0; i < cap(bb.blockers); i++ {
		bb.blockers <- struct{}{}
	}
	bb.Report()
}

// Stop halts the brute-force process.
func (bb *Barbarian) Stop() {
	close(bb.stop)
}

// Report updates the statistics after execution.
func (bb *Barbarian) Report() {
	bb.Stats.Finish = time.Now()
	bb.Stats.UpTime = bb.Stats.Finish.Sub(bb.Stats.Start)
	if bb.Stats.UpTime > 0 {
		bb.Stats.Execps = float64(bb.Stats.Execs) / bb.Stats.UpTime.Seconds()
	}
}
