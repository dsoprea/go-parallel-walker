package pathwalk

import (
	"fmt"
	"time"
)

// Stats describes all stats collected by the walking process.
type Stats struct {
	// JobsDispatchedToNewWorker is the number of workers that were started to
	// process a job.
	JobsDispatchedToNewWorker int

	// JobsDispatchedToIdleWorker is the number of jobs that were dispatched to
	// an available, idle worker rather than starting a new one.
	JobsDispatchedToIdleWorker int

	// FilesVisited is the number of files that were visited.
	FilesVisited int

	// DirectoriesVisited is the number of directories that were visited.
	DirectoriesVisited int

	// EntryBatchesProcessed is the number of batches that directory entries
	// were parceled into while processing.
	EntryBatchesProcessed int

	// IdleWorkerTime is the duration of all between-job time spent by workers.
	// Only includes time between jobs and time between last job and timeout
	// (leading to shutdown). Does not include time between the last job and a
	// closed channel being detected (which is not true idleness).
	IdleWorkerTime time.Duration

	// DirectoriesIgnored is the number of directories that were signaled to be
	// skipped using `ErrSkipDirectory`.
	DirectoriesIgnored int

	// PathFilterIncludes is the number of path include hits or exclude misses
	// if at least one filter rule was provided.
	PathFilterIncludes int

	// PathFilterExcludes is the number of path include misses or exclude hits
	// if at least one filter rule was provided.
	PathFilterExcludes int

	// FileFilterIncludes is the number of file include hits or exclude misses
	// if at least one filter rule was provided.
	FileFilterIncludes int

	// FileFilterExcludes is the number of file include misses or exclude hits
	// if at least one filter rule was provided.
	FileFilterExcludes int
}

// Dump prints all statistics.
func (stats Stats) Dump() {
	fmt.Printf("Processing Statistics\n")
	fmt.Printf("=====================\n")

	fmt.Printf("JobsDispatchedToNewWorker: (%d)\n", stats.JobsDispatchedToNewWorker)
	fmt.Printf("JobsDispatchedToIdleWorker: (%d)\n", stats.JobsDispatchedToIdleWorker)
	fmt.Printf("FilesVisited: (%d)\n", stats.FilesVisited)
	fmt.Printf("DirectoriesVisited: (%d)\n", stats.DirectoriesVisited)
	fmt.Printf("EntryBatchesProcessed: (%d)\n", stats.EntryBatchesProcessed)
	fmt.Printf("IdleWorkerTime: (%.03f) seconds\n", float64(stats.IdleWorkerTime)/float64(time.Second))
	fmt.Printf("DirectoriesIgnored: (%d)\n", stats.DirectoriesIgnored)
	fmt.Printf("PathFilterIncludes: (%d)\n", stats.PathFilterIncludes)
	fmt.Printf("PathFilterExcludes: (%d)\n", stats.PathFilterExcludes)
	fmt.Printf("FileFilterIncludes: (%d)\n", stats.FileFilterIncludes)
	fmt.Printf("FileFilterExcludes: (%d)\n", stats.FileFilterExcludes)

	fmt.Printf("\n")
}
