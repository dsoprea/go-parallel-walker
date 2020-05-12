package pathwalk

import (
	"time"
)

const (
	// defaultConcurrency is the default number of workers allowed to run in
	// parallel. This needs to accommodate the intermediate batching that occurs
	// as directories are chunked into jobs as well as the workers that call
	// the user callback for individual folders and files (one goroutine calls
	// one callback). Otherwise, you will experience deadlocks.
	//
	// In our testing, sometimes a lower number than 400 had higher performance,
	// but a) frequently hung on the unit-tests that simulate real-world
	// directories (large size with varying depth), which is not a good sign,
	// and b) it's hard to determine a proper default value because it's gonna
	// be tied to the queue-size and batch-size and directory size, and the
	// latter will typically stagger greatly for the average case. We ran into
	// additional problems with large paths, where perhaps the number of
	// directories and the order they were encountered and the likelihood of
	// them being encountered more frequently or faster than actual files, led
	// to us dead-locking unless we raised the default concurrency.
	defaultConcurrency = 400

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
