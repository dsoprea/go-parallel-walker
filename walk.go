package pathwalk

import (
	"errors"
	"io"
	"os"
	"path"
	"reflect"
	"sync"
	"time"

	"github.com/dsoprea/go-logging"
)

var (
	walkLogger = log.NewLogger("pathwalk.walk")
)

const (
	// defaultConcurrency is the default number of workers allowed to run in
	// parallel. This needs to accommodate the intermediate batching that occurs
	// as directories are chunked into jobs as well as the workers that call
	// the user callback for individual folders and files (one goroutine calls
	// one callback). Otherwise, you will experience deadlocks.
	//
	// In our testing, sometimes a lower number than 200 had higher performance,
	// but a) frequently hung on the unit-tests that simulate real-world
	// directories (large size with varying depth), which is not a good sign,
	// and b) it's hard to determine a proper default value because it's gonna
	// be tied to the queue-size and batch-size and directory size, and the
	// latter will typically stagger greatly for the average case.
	defaultConcurrency = 200

	// defaultBufferSize is the default size of the job channel.
	defaultBufferSize = 1000

	// defaultDirectoryEntryBatchSize is the default parcel size that we chunk
	// directory entries into before individually dispatching them for handling.
	defaultDirectoryEntryBatchSize = 100

	// defaultTimeoutDuration is the amount of time that can elapsed without any
	// activity before we timeout and complain about dead-lock.
	defaultTimeoutDuration = time.Second * 1

	// maxWorkerIdleDuration is how long a work waits while idle for new jobs
	// before it shuts down.
	maxWorkerIdleDuration = time.Second * 2

	// workerIdleCheckInterval is how often the worker will check if it's idle
	// and how long it has been.
	workerIdleCheckInterval = time.Second * 2

	// frontendIdleCheckInterval is how often the frontend checks for the find
	// to be done.
	frontendIdleCheckInterval = time.Millisecond * 500
)

var (
	// ErrSkipDirectory can be returned by the visitor if a directory to skip
	// walking its contents.
	ErrSkipDirectory = errors.New("skip directory")
)

// WalkFunc is the function type for the callback.
type WalkFunc func(parentPath string, info os.FileInfo) (err error)

// Walk knows how to traverse a tree in parallel.
type Walk struct {
	rootPath string

	concurrency     int
	bufferSize      int
	batchSize       int
	timeoutDuration time.Duration

	jobsC   chan Job
	errorsC chan error
	wg      *sync.WaitGroup

	workerCount     int
	idleWorkerCount int
	stateLocker     sync.Mutex

	walkFunc WalkFunc

	jobsInFlight  int
	counterLocker sync.Mutex

	stats       Stats
	statsLocker sync.Mutex

	// hasFinished indicates that al jobs have been processed.
	hasFinished bool

	// hasStopped indicates that workers should no longer be running.
	hasStopped bool

	filter           internalFilter
	doLogFilterStats bool
}

// NewWalk returns a new Walk struct.
func NewWalk(rootPath string, walkFunc WalkFunc) (walk *Walk) {
	walk = &Walk{
		rootPath: rootPath,

		concurrency:     defaultConcurrency,
		bufferSize:      defaultBufferSize,
		batchSize:       defaultDirectoryEntryBatchSize,
		timeoutDuration: defaultTimeoutDuration,

		walkFunc: walkFunc,
	}

	// Initialize empty filter state.

	filter := Filter{}
	walk.SetFilter(filter)

	return walk
}

// SetFilter sets filtering parameters for the next call to Run(). Behavior is
// undefined if this is changed *during* a call to `Run()`. The filters will be
// sorted automatically.
func (walk *Walk) SetFilter(filter Filter) {
	walk.filter = newInternalFilter(filter)

	// Only log the stats if we have any filters.
	walk.doLogFilterStats =
		len(walk.filter.includePaths) > 0 ||
			len(walk.filter.excludePaths) > 0 ||
			len(walk.filter.includeFilenames) > 0 ||
			len(walk.filter.excludeFilenames) > 0
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

	// Intentionally does not set `hasFinished`, as we are specifically aborting
	// the process.
}

// SetConcurrency sets an alternative maximum number of workers.
func (walk *Walk) SetConcurrency(concurrency int) {
	walk.concurrency = concurrency
}

// SetBufferSize sets an alternative size for the job channel.
func (walk *Walk) SetBufferSize(bufferSize int) {
	walk.bufferSize = bufferSize
}

// SetBatchSize sets an alternative size for the parcels of directory entries
// dispatched into jobs.
func (walk *Walk) SetBatchSize(batchSize int) {
	walk.batchSize = batchSize
}

// SetGlobalTimeoutDuration sets a non-default duration, after which if no
// activity has happened than we should consider ourselves dead-locked.
func (walk *Walk) SetGlobalTimeoutDuration(timeoutDuration time.Duration) {
	walk.timeoutDuration = timeoutDuration
}

// InitSync sets-up the synchronization state. This is isolated as a separate
// step to support testing.
func (walk *Walk) InitSync() {
	// Our jobs channel.
	walk.jobsC = make(chan Job, walk.concurrency)

	// Our error channel
	walk.errorsC = make(chan error, 0)

	// Allows us to wait until jobs have completed before we exit.
	walk.wg = new(sync.WaitGroup)

	// To facilitate reuse of the struct for follow-up operations.
	walk.jobsInFlight = 0

	walk.stats = Stats{}
	walk.hasFinished = false
	walk.hasStopped = false
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
		walk.stateLocker.Lock()
		hasWorkers := walk.workerCount > 0 || walk.idleWorkerCount > 0
		walk.stateLocker.Unlock()

		if hasWorkers == true {
			// Wait/cleanup workers.

			isRunning := true
			var workerError error

			tick := time.NewTicker(frontendIdleCheckInterval)
			lastState := [2]int{0, 0}
			lastStateChange := time.Now()

			for isRunning == true {
				select {
				case err := <-walk.errorsC:
					workerError = err
					isRunning = false

					// Signals workers to close.
					close(walk.jobsC)
				case <-tick.C:
					// The same locker used to update this field.
					walk.counterLocker.Lock()
					isRunning = walk.hasStopped == false
					walk.counterLocker.Unlock()

					// Check for deadlock.

					currentState := [2]int{walk.stats.FilesVisited, walk.stats.DirectoriesVisited}

					if currentState != lastState {
						lastState = currentState
						lastStateChange = time.Now()
					} else if time.Since(lastStateChange) > walk.timeoutDuration {
						walk.errorsC <- errors.New("walk appears to be dead-locked; if this is not the case, provide a higher timeout duration")
					}
				}
			}

			tick.Stop()

			// The workers will terminate on their own, either because the count of
			// in-flight jobs has dropped to zero or the workers all starve and
			// terminate (which should never happen unless the concurrency level is
			// too high).
			walk.wg.Wait()

			close(walk.errorsC)

			if workerError != nil {
				log.Panicf("worker terminated under error: %s", workerError.Error())
			}
		}
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
	defer walk.stateLocker.Unlock()

	walk.idleWorkerCount++
}

// idleWorkerTickDown states that one worker is no longer idle.
func (walk *Walk) idleWorkerTickDown() {
	walk.stateLocker.Lock()
	defer walk.stateLocker.Unlock()

	walk.idleWorkerCount--
}

// nodeWorker represents one worker goroutine. It will process jobs, it will
// declare when it's idle (waiting for a job), and it'll eventually shutdown if
// it doesn't get any jobs.
func (walk *Walk) nodeWorker() {
	defer func() {
		if state := recover(); state != nil {
			err := log.Wrap(state.(error))
			log.PrintErrorf(err, "Node worker panicked.")

			walk.errorsC <- err
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
		walk.hasStopped = true
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
		if err != nil {
			walkLogger.Warningf(nil, "can not stat [%s]; it will be skipped: [%s]", path, err.Error())
			return nil
		}

		if info.IsDir() == true {
			jdn := newJobDirectoryNode(parentNodePath, info)

			err := walk.pushJob(jdn)
			log.PanicIf(err)
		} else if jdcb.DoProcessFiles() == true {
			// We'll only descend on a non-root path if it passed the path-
			// filter above.
			if walk.filter.IsFileIncluded(childFilename) != true {
				walkLogger.Debugf(nil, "File excluded: [%s]", childFilename)

				walk.statsFileFilterExcludeTickUp()
				continue
			}

			walk.statsFileFilterIncludeTickUp()

			jfn := newJobFileNode(parentNodePath, info)

			err := walk.pushJob(jfn)
			log.PanicIf(err)
		}
	}

	return nil
}

func (walk *Walk) statsPathFilterIncludeTickUp() {
	if walk.doLogFilterStats == false {
		return
	}

	walk.statsLocker.Lock()
	defer walk.statsLocker.Unlock()

	walk.stats.PathFilterIncludes++
}

func (walk *Walk) statsPathFilterExcludeTickUp() {
	if walk.doLogFilterStats == false {
		return
	}

	walk.statsLocker.Lock()
	defer walk.statsLocker.Unlock()

	walk.stats.PathFilterExcludes++
}

func (walk *Walk) statsFileFilterIncludeTickUp() {
	if walk.doLogFilterStats == false {
		return
	}

	walk.statsLocker.Lock()
	defer walk.statsLocker.Unlock()

	walk.stats.FileFilterIncludes++
}

func (walk *Walk) statsFileFilterExcludeTickUp() {
	if walk.doLogFilterStats == false {
		return
	}

	walk.statsLocker.Lock()
	defer walk.statsLocker.Unlock()

	walk.stats.FileFilterExcludes++
}

// handleJobDirectoryNode handles one directory note. It will read and parcel
// child files and directories.
func (walk *Walk) handleJobDirectoryNode(jdn jobDirectoryNode) (err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	walk.statsLocker.Lock()
	walk.stats.DirectoriesVisited++
	walk.statsLocker.Unlock()

	parentNodePath := jdn.ParentNodePath()
	info := jdn.Info()

	fqPath := path.Join(parentNodePath, info.Name())
	rootPathPrefixLen := len(walk.rootPath) + 1
	isIncluded := true
	if len(fqPath) > rootPathPrefixLen {
		relPath := fqPath[rootPathPrefixLen:]

		// We process every directory due to recursive filter support (the path
		// filters we are given apply to the complete parent expression), meaning
		// that we need to descend all of the way down through the tree in order to
		// know what is really included by the filters. However, we won't process
		// any files unless their parent directory mtch the filter (or there was no
		// filter).

		if walk.filter.IsPathIncluded(relPath) != true {
			walkLogger.Debugf(nil, "Directory excluded: [%s]", relPath)

			walk.statsPathFilterExcludeTickUp()
			isIncluded = false
		} else {
			walk.statsPathFilterIncludeTickUp()
		}
	}

	if isIncluded {
		// Call callback, but only if it didn't get excluded by the filter.

		// We don't concern ourselves with symlinked directories. If they don't want
		// to descend into them, they can detect them and skip.

		err = walk.walkFunc(parentNodePath, info)
		if err != nil {
			if err == ErrSkipDirectory {
				walk.statsLocker.Lock()
				walk.stats.DirectoriesIgnored++
				walk.statsLocker.Unlock()

				return nil
			}

			log.Panic(err)
		}
	}

	// Now, push jobs for directory children.

	path := path.Join(parentNodePath, info.Name())

	f, err := os.Open(path)
	log.PanicIf(err)

	defer f.Close()

	batchNumber := 0
	for {
		names, err := f.Readdirnames(walk.batchSize)
		if err != nil {
			if err == io.EOF {
				break
			}

			log.Panic(err)
		}

		jdcb := newJobDirectoryContentsBatch(path, batchNumber, names, isIncluded)

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
