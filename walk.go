package pathwalk

import (
	"io"
	"os"
	"path"
	"reflect"
	"sync"
	"time"

	"github.com/dsoprea/go-logging"
)

const (
	// defaultConcurrency is the number of workers allowed to run in parallel.
	// This is an untested number, but needs to accommodate the intermediate
	// batching that occurs as directories are chunked into jobs as well as the
	// workers that call the user callback for individual folders and files (one
	// goroutine calls one callback).
	defaultConcurrency = 200

	// defaultBufferSize is the default size of the job channel.
	defaultBufferSize = 1000

	// maxWorkerIdleDuration is how long a work waits while idle for new jobs
	// before it shuts down.
	maxWorkerIdleDuration = time.Second * 2

	// workerIdleCheckInterval is how often the worker will check if it's idle
	// and how long it has been.
	workerIdleCheckInterval = time.Second * 2

	// directoryEntryBatchSize is the parcel size that we chunk directory
	// entries into before individually dispatching them for handling.
	directoryEntryBatchSize = 100
)

// WalkFunc is the function type for the callback.
type WalkFunc func(parentPath string, info os.FileInfo) (err error)

// Walk knows how to traverse a tree in parallel.
type Walk struct {
	rootPath    string
	concurrency int
	bufferSize  int

	jobsC chan Job
	wg    *sync.WaitGroup

	workerCount     int
	idleWorkerCount int
	stateLocker     sync.Mutex

	walkFunc WalkFunc

	jobsInFlight  int
	counterLocker sync.Mutex

	stats       Stats
	statsLocker sync.Mutex

	hasFinished bool

	filters internalFilters
}

// NewWalk returns a new Walk struct.
func NewWalk(rootPath string, walkFunc WalkFunc) *Walk {
	return &Walk{
		rootPath:    rootPath,
		concurrency: defaultConcurrency,
		bufferSize:  defaultBufferSize,
		walkFunc:    walkFunc,
	}
}

// SetFilters sets filtering parameters for the next call to Run(). Behavior is
// undefined if this is changed *during* a call to `Run()`. The filters will be
// sorted automatically.
func (walk *Walk) SetFilters(filters Filters) {
	walk.filters = newInternalFilters(filters)
}

// Stats prints statistics about the last walking operation.
func (walk *Walk) Stats() Stats {
	return walk.stats
}

// HasFinished returns whether all entries have been visited and processed.
func (walk *Walk) HasFinished() bool {
	return walk.hasFinished
}

// Stop will signal all of the workers to terminate if Run() has not yet
// returned. This is provided for the user to call as a result of some logic in
// the callback that calls for immediate return.
func (walk *Walk) Stop() {
	close(walk.jobsC)

	// Intentionally does not set `hasFinished`.
}

// SetConcurrency sets an alternative maximum number of workers.
func (walk *Walk) SetConcurrency(concurrency int) {
	walk.concurrency = concurrency
}

// SetBufferSize sets an alternative size for the job channel.
func (walk *Walk) SetBufferSize(bufferSize int) {
	walk.bufferSize = bufferSize
}

// InitSync sets-up the synchronization state. This is isolated as a separate
// step to support testing.
func (walk *Walk) InitSync() {
	// Our job pipeline.
	walk.jobsC = make(chan Job, walk.concurrency)

	// Allows us to wait until jobs have completed before we exit.
	walk.wg = new(sync.WaitGroup)

	// To facilitate reuse of the struct for follow-up operations.
	walk.jobsInFlight = 0

	walk.stats = Stats{}
	walk.hasFinished = false
}

// Run forks workers to process the tree. All workers will have quit by the time we return.
func (walk *Walk) Run() (err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	walk.InitSync()

	defer func() {
		// Wait/cleanup workers.

		// The workers will terminate on their own, either because the count of
		// in-flight jobs has dropped to zero or the workers all starve and
		// terminate (which should never happen unless the concurrency level is
		// too high).
		walk.wg.Wait()
	}()

	info, err := os.Stat(walk.rootPath)
	log.PanicIf(err)

	parentPath := path.Dir(walk.rootPath)
	initialJob := newJobDirectoryNode(parentPath, info)

	err = walk.pushJob(initialJob)
	log.PanicIf(err)

	return nil
}

// pushJob queues a job. It'll start a new worker if there are no existing ones
// or there are but none are idle and we're under-capacity.
//
// This function (and, thus, the path walk) will throttle on channel capacity.
func (walk *Walk) pushJob(job Job) (err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	walk.stateLocker.Lock()
	canStart := walk.idleWorkerCount <= 0 && walk.workerCount < walk.concurrency
	walk.stateLocker.Unlock()

	// All workers are occupied but we can start another one.
	if canStart {
		walk.stateLocker.Lock()

		walk.workerCount++
		walk.wg.Add(1)

		walk.stateLocker.Unlock()

		go walk.nodeWorker()
	} else {
		walk.statsLocker.Lock()
		walk.stats.JobsDispatchedToIdleWorker++
		walk.statsLocker.Unlock()
	}

	walk.jobTickUp()

	// Here, a job gets pushed whether any workers are idle or not.
	walk.jobsC <- job

	return nil
}

// idleWorkerTickUp states that one worker has become idle.
func (walk *Walk) idleWorkerTickUp() {
	walk.stateLocker.Lock()
	walk.idleWorkerCount++
	walk.stateLocker.Unlock()
}

// idleWorkerTickDown states that one worker is no longer idle.
func (walk *Walk) idleWorkerTickDown() {
	walk.stateLocker.Lock()
	walk.idleWorkerCount--
	walk.stateLocker.Unlock()
}

// nodeWorker represents one worker goroutine. It will process jobs, it will
// declare when it's idle (waiting for a job), and it'll eventually shutdown if
// it doesn't get any jobs.
func (walk *Walk) nodeWorker() {
	defer func() {
		if state := recover(); state != nil {
			err := log.Wrap(state.(error))
			log.PrintErrorf(err, "Node worker panicked.")
		}
	}()

	isWorking := false
	tick := time.NewTicker(workerIdleCheckInterval)

	walk.statsLocker.Lock()
	walk.stats.JobsDispatchedToNewWorker++
	walk.statsLocker.Unlock()

	defer func() {
		tick.Stop()

		walk.stateLocker.Lock()

		// If the idle-worker-count was decremented but something prevented us
		// from incrementing it again. There could've been an error. This makes
		// sure that the arithmetic isn't off.
		if isWorking == true {
			walk.idleWorkerCount++
		}

		walk.workerCount--
		walk.idleWorkerCount--

		walk.stateLocker.Unlock()

		walk.wg.Done()
	}()

	lastActivityTime := time.Now()

	walk.idleWorkerTickUp()

	for {
		select {
		case job, ok := <-walk.jobsC:
			if ok == false {
				// Channel is closed. The application must be closing. Shutdown.

				return
			}

			walk.idleWorkerTickDown()

			walk.statsLocker.Lock()
			walk.stats.IdleWorkerTime += time.Since(lastActivityTime)
			walk.statsLocker.Unlock()

			// This helps us manage our state if there's a panic.
			isWorking = true

			lastActivityTime = time.Now()

			err := walk.handleJob(job)
			log.PanicIf(err)

			isWorking = false

			walk.idleWorkerTickUp()
		case <-tick.C:
			if isWorking == false && time.Since(lastActivityTime) > maxWorkerIdleDuration {
				// We haven't had anything to do for a while. Shutdown.

				walk.statsLocker.Lock()
				walk.stats.IdleWorkerTime += time.Since(lastActivityTime)
				walk.statsLocker.Unlock()

				return
			}
		}
	}

	// Execution will reach here before hitting the defer and cleaning up.
}

// handleJob handles one queued job.
func (walk *Walk) handleJob(job Job) (err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	switch t := job.(type) {
	case jobDirectoryContentsBatch:

		err := walk.handleJobDirectoryContentsBatch(t)
		log.PanicIf(err)

	case jobDirectoryNode:
		err := walk.handleJobDirectoryNode(t)
		log.PanicIf(err)

	case jobFileNode:
		err := walk.handleJobFileNode(t)
		log.PanicIf(err)

	default:
		log.Panicf("job not valid: [%v]", reflect.TypeOf(t))
	}

	walk.jobTickDown()

	return nil
}

func (walk *Walk) jobTickUp() {
	walk.counterLocker.Lock()
	defer walk.counterLocker.Unlock()

	walk.jobsInFlight++
}

func (walk *Walk) jobTickDown() {
	walk.counterLocker.Lock()
	defer walk.counterLocker.Unlock()

	walk.jobsInFlight--

	// Safety check.
	if walk.jobsInFlight < 0 {
		log.Panicf("job counter is unbalanced: (%d)", walk.jobsInFlight)
	}

	if walk.jobsInFlight <= 0 {
		close(walk.jobsC)
		walk.hasFinished = true
	}
}

// handleJobDirectoryContentsBatch processes a batch of N directory entries. We
// don't yet know whether they are files or directories.
func (walk *Walk) handleJobDirectoryContentsBatch(jdcb jobDirectoryContentsBatch) (err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	// Produce N leaf jobs from a batch of N items.

	parentNodePath := jdcb.ParentNodePath()
	for _, childFilename := range jdcb.ChildBatch() {
		path := path.Join(parentNodePath, childFilename)

		info, err := os.Stat(path)
		log.PanicIf(err)

		if info.IsDir() == true {

			// TODO(dustin): Apply directory filters.

			jdn := newJobDirectoryNode(parentNodePath, info)

			err := walk.pushJob(jdn)
			log.PanicIf(err)
		} else {

			// TODO(dustin): Apply file filters.

			jfn := newJobFileNode(parentNodePath, info)

			err := walk.pushJob(jfn)
			log.PanicIf(err)
		}
	}

	return nil
}

// handleJobDirectoryNode handles one directory note. It will read and parcel
// child files and directories.
func (walk *Walk) handleJobDirectoryNode(jdn jobDirectoryNode) (err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	// TODO(dustin): This should be able to return a specific error to skip child processing.

	walk.statsLocker.Lock()
	walk.stats.DirectoriesVisited++
	walk.statsLocker.Unlock()

	parentNodePath := jdn.ParentNodePath()
	info := jdn.Info()

	// We don't concern ourselves with symlinked directories. If they don't want
	// to descend into them, they can detect them and skip.
	err = walk.walkFunc(parentNodePath, info)
	log.PanicIf(err)

	// Now, push jobs for directory children.

	path := path.Join(parentNodePath, info.Name())

	f, err := os.Open(path)
	log.PanicIf(err)

	defer f.Close()

	batchNumber := 0
	for {
		names, err := f.Readdirnames(directoryEntryBatchSize)
		if err != nil {
			if err == io.EOF {
				break
			}

			log.Panic(err)
		}

		jdcb := newJobDirectoryContentsBatch(path, batchNumber, names)

		err = walk.pushJob(jdcb)
		log.PanicIf(err)

		batchNumber++
	}

	walk.statsLocker.Lock()
	walk.stats.EntryBatchesProcessed += batchNumber
	walk.statsLocker.Unlock()

	return nil
}

// handleJobFileNode handles one file node. This is a leaf operation.
func (walk *Walk) handleJobFileNode(jfn jobFileNode) (err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	walk.statsLocker.Lock()
	walk.stats.FilesVisited++
	walk.statsLocker.Unlock()

	parentNodePath := jfn.ParentNodePath()
	info := jfn.Info()

	err = walk.walkFunc(parentNodePath, info)
	log.PanicIf(err)

	return nil
}
