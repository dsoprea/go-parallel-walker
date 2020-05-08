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

// TODO(dustin): Make sure to handle symlinks (including directories that need to be descended).
// TODO(dustin): !! Finish adding documentation.

type WalkFunc func(parentPath string, info os.FileInfo) (err error)

type Walk struct {
	rootPath        string
	concurrency     int
	jobsC           chan Job
	wg              *sync.WaitGroup
	workerCount     int
	idleWorkerCount int
	walkFunc        WalkFunc
	stateLocker     sync.Mutex
}

func NewWalk(rootPath string, walkFunc WalkFunc) *Walk {
	return &Walk{
		rootPath:    rootPath,
		concurrency: defaultConcurrency,
		walkFunc:    walkFunc,
	}
}

func (walk *Walk) SetConcurrency() {
	walk.concurrency = defaultConcurrency
}

func (walk *Walk) initSync() {
	// Our job pipeline.
	walk.jobsC = make(chan Job, walk.concurrency)

	// Allows us to wait until jobs have completed before we exit.
	walk.wg = new(sync.WaitGroup)
}

func (walk *Walk) Run() (err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	walk.initSync()

	defer func() {
		close(walk.jobsC)
		walk.wg.Wait()
	}()

	info, err := os.Stat(walk.rootPath)
	log.PanicIf(err)

	parentPath := path.Dir(walk.rootPath)
	initialJob := newJobDirectoryNode(parentPath, info)

	err = walk.pushJob(initialJob)
	log.PanicIf(err)

	// TODO(dustin): !! Wait for all of the workers to terminate. We can eliminate the wait by keep folder and file counters and then closing the channel at the bottom of the file handler if both are zero.
	walk.wg.Wait()
	close(walk.jobsC)

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
	}

	// Here, a job gets pushed whether any workers are idle or not.
	walk.jobsC <- job

	return nil
}

// nodeWorker represents one worker goroutine. It will process jobs, it will
// declare when it's idle (waiting for a job), and it'll eventually shutdown if
// it doesn't get any jobs.
func (walk *Walk) nodeWorker() {
	isWorking := false
	tick := time.NewTicker(workerIdleCheckInterval)

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

	// TODO(dustin): Still needs a way to report errors.

	lastActivityTime := time.Now()

	walk.stateLocker.Lock()
	walk.idleWorkerCount++
	walk.stateLocker.Unlock()

	for {
		select {
		case job, ok := <-walk.jobsC:
			if ok == false {
				// Channel is closed. The application must be closing. Shutdown.
				// TODO(dustin): !! Debugging.
				return
			}

			walk.stateLocker.Lock()
			walk.idleWorkerCount--
			walk.stateLocker.Unlock()

			// This helps us manage our state if there's a panic.
			isWorking = true

			lastActivityTime = time.Now()

			err := walk.handleJob(job)
			log.PanicIf(err)

			isWorking = false

			walk.stateLocker.Lock()
			walk.idleWorkerCount++
			walk.stateLocker.Unlock()
		case <-tick.C:
			if isWorking == false && time.Since(lastActivityTime) > maxWorkerIdleDuration {
				// We haven't had anything to do for a while. Shutdown.
				return
			}
		}
	}

	// Execution will reach here before hitting the defer and cleaning up.
}

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

	return nil
}

// handleJobDirectoryContentsBatch processes a batch of N directory entries. We
// don't yet know whether they are files or directories.
func (walk *Walk) handleJobDirectoryContentsBatch(jdcb jobDirectoryContentsBatch) (err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	// TODO(dustin): !! Apply directory filters

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

func (walk *Walk) handleJobDirectoryNode(jdn jobDirectoryNode) (err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	// TODO(dustin): This should be able to return a specific error to skip child processing.

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

	for {
		names, err := f.Readdirnames(directoryEntryBatchSize)
		if err != nil {
			if err == io.EOF {
				break
			}

			log.Panic(err)
		}

		jdcb := newJobDirectoryContentsBatch(path, names)

		err = walk.pushJob(jdcb)
		log.PanicIf(err)
	}

	return nil
}

func (walk *Walk) handleJobFileNode(jfn jobFileNode) (err error) {
	defer func() {
		if state := recover(); state != nil {
			err = log.Wrap(state.(error))
		}
	}()

	parentNodePath := jfn.ParentNodePath()
	info := jfn.Info()

	err = walk.walkFunc(parentNodePath, info)
	log.PanicIf(err)

	return nil
}
