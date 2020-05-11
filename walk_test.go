package pathwalk

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"io/ioutil"

	"github.com/dsoprea/go-logging"
	"github.com/dsoprea/go-utility/filesystem"
	"github.com/gobwas/glob"
)

func TestWalk_nodeWorker__openAndClose(t *testing.T) {
	wg := new(sync.WaitGroup)
	wg.Add(1)

	jobsC := make(chan Job, 1)

	walk := &Walk{
		workerCount: 1,
		wg:          wg,
		jobsC:       jobsC,
	}

	finished := false

	// Synchronize to make the race-detector happy.
	m := sync.Mutex{}

	go func() {
		walk.nodeWorker()

		m.Lock()
		defer m.Unlock()

		finished = true
	}()

	close(jobsC)
	wg.Wait()

	time.Sleep(time.Millisecond * 100)

	m.Lock()

	if finished != true {
		m.Unlock()

		t.Fatalf("Worker did not exit.")
	}

	m.Unlock()
}

func TestWalk_nodeWorker__closeWhenIdle(t *testing.T) {

	// Our idle-timeout is very short. This test creates a worker adn waits for it to quick soon after.

	wg := new(sync.WaitGroup)
	wg.Add(1)

	jobsC := make(chan Job, 1)

	walk := &Walk{
		workerCount: 1,
		wg:          wg,
		jobsC:       jobsC,
	}

	finished := false

	// Synchronize to make the race-detector happy.
	m := sync.Mutex{}

	go func() {
		walk.nodeWorker()

		m.Lock()
		defer m.Unlock()

		finished = true
	}()

	wg.Wait()
	close(jobsC)

	// Give the wrapper function above a moment to cleanup.
	time.Sleep(time.Millisecond * 100)

	m.Lock()

	if finished != true {
		m.Unlock()

		t.Fatalf("Worker did not exit.")
	}

	m.Unlock()
}

func TestWalk_nodeWorker__processOneJob(t *testing.T) {

	// Our idle-timeout is very short. This test creates a worker adn waits for it to quick soon after.

	wg := new(sync.WaitGroup)
	wg.Add(1)

	jobsC := make(chan Job, 1)

	var handledFilename string

	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		handledFilename = info.Name()

		// Stop the test ASAP.
		close(jobsC)

		return nil
	}

	walk := &Walk{
		workerCount: 1,
		wg:          wg,
		jobsC:       jobsC,
		walkFunc:    walkFunc,
	}

	// We'll have exactly one job in-flight, but we also want another one to
	// prevent the one job from inducing the worker to be closed.
	walk.jobsInFlight = 2

	finished := false

	// Synchronize to make the race-detector happy.
	m := sync.Mutex{}

	go func() {
		walk.nodeWorker()

		m.Lock()
		defer m.Unlock()

		finished = true
	}()

	testFileInfo := rifs.NewSimpleFileInfoWithFile("test.file", 0, 0, time.Time{})
	jobsC <- newJobFileNode("", testFileInfo)

	wg.Wait()

	// Give the wrapper function above a moment to cleanup.
	time.Sleep(time.Millisecond * 100)

	m.Lock()

	if finished != true {
		m.Unlock()

		t.Fatalf("Worker did not exit.")
	}

	m.Unlock()

	if handledFilename != "test.file" {
		t.Fatalf("Job was not processed.")
	}
}

func TestWalk_nodeWorker__processMultipleJob(t *testing.T) {

	// Our idle-timeout is very short. This test creates a worker adn waits for it to quick soon after.

	wg := new(sync.WaitGroup)
	wg.Add(1)

	jobsC := make(chan Job, 1)

	handledFilenames := make([]string, 0)

	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		handledFilename := info.Name()
		handledFilenames = append(handledFilenames, handledFilename)

		return nil
	}

	walk := &Walk{
		workerCount: 1,
		wg:          wg,
		jobsC:       jobsC,
		walkFunc:    walkFunc,
	}

	// Give it the right count of jobs so that the last one will automatically
	// close the channel.
	walk.jobsInFlight = 6

	finished := false

	// Synchronize to make the race-detector happy.
	m := sync.Mutex{}

	go func() {
		walk.nodeWorker()

		m.Lock()
		defer m.Unlock()

		finished = true
	}()

	// The timeout is two seconds but we'll pump in a new job at one-second
	// intervals. This will test that its idle tracking is working and that the
	// handler is triggering.
	for i := 0; i < 6; i++ {
		filename := fmt.Sprintf("test-%d.file", i)
		oneTestFileInfo := rifs.NewSimpleFileInfoWithFile(filename, 0, 0, time.Time{})
		jobsC <- newJobFileNode("", oneTestFileInfo)
		time.Sleep(time.Second * 1)
	}

	wg.Wait()

	// Give the wrapper function above a moment to cleanup.
	time.Sleep(time.Millisecond * 100)

	m.Lock()

	if finished != true {
		m.Unlock()

		t.Fatalf("Worker did not exit.")
	}

	m.Unlock()

	expectedFiles := []string{
		"test-0.file",
		"test-1.file",
		"test-2.file",
		"test-3.file",
		"test-4.file",
		"test-5.file",
	}

	if reflect.DeepEqual(handledFilenames, expectedFiles) != true {
		t.Fatalf("Jobs were not processed correctly: %v", handledFilenames)
	}
}

func TestWalk_nodeWorker__pushJob__closeImmediately(t *testing.T) {
	var handledFilename string

	var walk *Walk

	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		handledFilename = info.Name()

		// Stop the test ASAP.
		close(walk.jobsC)

		return nil
	}

	walk = NewWalk("", walkFunc)

	walk.InitSync()

	// Keep the coutner one higher than the actual count so that the last job
	// won't induce the worker to be closed.
	walk.jobsInFlight = 1

	testFileInfo := rifs.NewSimpleFileInfoWithFile("test.file", 0, 0, time.Time{})
	jfn := newJobFileNode("", testFileInfo)

	walk.pushJob(jfn)
	walk.wg.Wait()

	if handledFilename != "test.file" {
		t.Fatalf("Job was not processed.")
	}
}

func TestWalk_nodeWorker__pushJob__closeWhenIdle(t *testing.T) {
	var handledFilename string

	var walk *Walk

	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		handledFilename = info.Name()
		return nil
	}

	walk = NewWalk("", walkFunc)

	walk.InitSync()

	// We'll state that there's one more job than there actually is so that the
	// job-count won't drop all of the way to zero and the channel won't be
	// automatically closed.
	walk.jobsInFlight++

	testFileInfo := rifs.NewSimpleFileInfoWithFile("test.file", 0, 0, time.Time{})
	jfn := newJobFileNode("", testFileInfo)

	walk.pushJob(jfn)
	walk.wg.Wait()
	close(walk.jobsC)

	if handledFilename != "test.file" {
		t.Fatalf("Job was not processed.")
	}
}

func TestWalk_handleJobDirectoryNode(t *testing.T) {
	// Stage a directory to walk.

	fileCount := directoryEntryBatchSize * 3
	tempPath, tempFilenames := FillFlatTempPath(fileCount, []string{"testdir"})

	defer func() {
		os.RemoveAll(tempPath)
	}()

	tempFilenames = append(tempFilenames, "testdir")
	tempFilenames.Sort()

	// Setup walk.

	m := sync.Mutex{}

	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		handledFilename := info.Name()

		m.Lock()
		defer m.Unlock()

		j := tempFilenames.Search(handledFilename)
		if j >= len(tempFilenames) || tempFilenames[j] != handledFilename {
			t.Fatalf("Handled file was not in the temporary-files list: [%s]", handledFilename)
		}

		tempFilenames = append(tempFilenames[:j], tempFilenames[j+1:]...)

		return nil
	}

	walk := NewWalk("", walkFunc)
	walk.InitSync()

	// Handle the root directory node.

	sfi := rifs.NewSimpleFileInfoWithDirectory("testdir", time.Time{})
	jdn := newJobDirectoryNode(tempPath, sfi)

	// This will fork workers to process the children in batches.
	err := walk.handleJobDirectoryNode(jdn)
	log.PanicIf(err)

	walk.wg.Wait()

	if len(tempFilenames) != 0 {
		fmt.Printf("One or more files were not handled:\n")
		for i, name := range tempFilenames {
			fmt.Printf("%d: %s\n", i, name)
		}

		fmt.Printf("\n")

		t.Fatalf("One or more files were not handled.")
	}
}

func TestWalk_handleJobDirectoryContentsBatch(t *testing.T) {
	// Stage a directory to walk.

	fileCount := 5
	tempPath, tempFilenames := FillFlatTempPath(fileCount, nil)

	defer func() {
		os.RemoveAll(tempPath)
	}()

	// Setup walk.

	m := sync.Mutex{}

	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		handledFilename := info.Name()

		m.Lock()
		defer m.Unlock()

		j := tempFilenames.Search(handledFilename)
		if j >= len(tempFilenames) || tempFilenames[j] != handledFilename {
			t.Fatalf("Handled file was not in the temporary-files list: [%s]", handledFilename)
		}

		tempFilenames = append(tempFilenames[:j], tempFilenames[j+1:]...)

		return nil
	}

	walk := NewWalk("", walkFunc)
	walk.InitSync()

	walk.jobsInFlight = 5

	// Handle the root directory node.

	// We copy the slice going in as the argument to prevent races with the
	// modification in the callback above.
	childBatch := make([]string, len(tempFilenames))
	copy(childBatch, tempFilenames)

	jdcb := newJobDirectoryContentsBatch(tempPath, 0, childBatch, true)

	// This will fork workers to process the children in batches.
	err := walk.handleJobDirectoryContentsBatch(jdcb)
	log.PanicIf(err)

	walk.wg.Wait()

	if len(tempFilenames) != 0 {
		t.Fatalf("Not all files were handled: %v", tempFilenames)
	}
}

func TestWalk_Run__simple(t *testing.T) {
	// Stage test directory.

	fileCount := 200
	tempPath, tempFilenames := FillFlatTempPath(fileCount, nil)

	tempFilenames = append(tempFilenames, path.Base(tempPath))
	tempFilenames.Sort()

	// Walk

	defer func() {
		os.RemoveAll(tempPath)
	}()

	m := sync.Mutex{}

	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		handledFilename := info.Name()

		m.Lock()
		defer m.Unlock()

		j := tempFilenames.Search(handledFilename)
		if j >= len(tempFilenames) || tempFilenames[j] != handledFilename {
			t.Fatalf("Handled file was not in the temporary-files list: [%s]", handledFilename)
			return nil
		}

		tempFilenames = append(tempFilenames[:j], tempFilenames[j+1:]...)

		return nil
	}

	walk := NewWalk(tempPath, walkFunc)

	err := walk.Run()
	log.PanicIf(err)

	if len(tempFilenames) != 0 {
		t.Fatalf("Not all files were handled: %v", tempFilenames)
	} else if walk.HasFinished() != true {
		t.Fatalf("HasFinished() is not true.")
	}
}

func TestWalk_Run__terminateBecauseOfJobError(t *testing.T) {
	// This test makes sure that a job panic will terminate the pipeline (and
	// not just hang or casually exit with empty results).

	walk := NewWalk("/invalid/path", nil)

	err := walk.Run()
	if err == nil {
		t.Fatalf("Expected panic due to invalid path.")
	} else if err.Error() != "stat /invalid/path: no such file or directory" {
		log.Panic(err)
	}
}

func TestWalk_Run__simple__skip(t *testing.T) {
	// Stage test directory.

	tempPath, err := ioutil.TempDir("", "")
	log.PanicIf(err)

	err = os.Mkdir(path.Join(tempPath, "dir1"), 0755)
	log.PanicIf(err)

	err = ioutil.WriteFile(path.Join(tempPath, "dir1", "file1"), []byte{}, 0)
	log.PanicIf(err)

	err = os.Mkdir(path.Join(tempPath, "dir2"), 0755)
	log.PanicIf(err)

	err = ioutil.WriteFile(path.Join(tempPath, "dir2", "file2"), []byte{}, 0)
	log.PanicIf(err)

	err = os.Mkdir(path.Join(tempPath, "dir3"), 0755)
	log.PanicIf(err)

	err = ioutil.WriteFile(path.Join(tempPath, "dir3", "file3"), []byte{}, 0)
	log.PanicIf(err)

	// Walk

	defer func() {
		os.RemoveAll(tempPath)
	}()

	m := sync.Mutex{}

	tempFilenames := make([]string, 0)
	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		if info.IsDir() == true {
			if info.Name() == "dir2" {
				return ErrSkipDirectory
			}

			return nil
		}

		tempFilename := info.Name()

		m.Lock()
		defer m.Unlock()

		tempFilenames = append(tempFilenames, tempFilename)

		return nil
	}

	walk := NewWalk(tempPath, walkFunc)

	err = walk.Run()
	log.PanicIf(err)

	sort.Strings(tempFilenames)
	expectedHandled := []string{"file1", "file3"}

	if reflect.DeepEqual(tempFilenames, expectedHandled) != true {
		t.Fatalf("Handled files not correct: %v\n", tempFilenames)
	}
}

func TestWalk_Run__simpleReuse(t *testing.T) {
	// Test that we can run several walks on the same `Walk` struct.

	// Stage test directory.

	fileCount := 200
	tempPath, _ := FillFlatTempPath(fileCount, nil)

	defer func() {
		os.RemoveAll(tempPath)
	}()

	// Walk

	m := sync.Mutex{}

	visitCount := 0
	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		m.Lock()
		defer m.Unlock()

		visitCount++

		return nil
	}

	walk := NewWalk(tempPath, walkFunc)

	expectedCount := 0
	for i := 0; i < 3; i++ {
		walk.InitSync()

		err := walk.Run()
		log.PanicIf(err)

		// Number of test files plus the root path.
		expectedCount += fileCount + 1
	}

	if visitCount != expectedCount {
		t.Fatalf("visitCount is not correct: (%d) != (%d)", visitCount, expectedCount)
	}
}

func TestWalk_Run__heirarchical(t *testing.T) {
	// Stage test directory.

	fileCount := 500
	tempPath, tempFiles := FillHeirarchicalTempPath(fileCount, nil)

	// Build a big map of all of the directories that we expect to see.

	tempPaths := make(map[string]struct{})
	for _, relFilepath := range tempFiles {
		relPath := path.Dir(relFilepath)

		for ptr := relPath; ptr != "."; ptr = path.Dir(ptr) {
			tempPaths[ptr] = struct{}{}
		}
	}

	// Walk

	defer func() {
		os.RemoveAll(tempPath)
	}()

	m := sync.Mutex{}

	len_ := len(tempPath)
	tempPathName := path.Base(tempPath)
	rootNodeHit := false
	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		var relParentPath string
		if len(parentPath) > len_ {
			relParentPath = parentPath[len_+1:]
		} else if relParentPath == "" && info.Name() == tempPathName {
			// This is the root node. Ignore.
			rootNodeHit = true
			return nil
		}

		m.Lock()
		defer m.Unlock()

		if info.IsDir() == true {
			relPath := path.Join(relParentPath, info.Name())

			if _, found := tempPaths[relPath]; found != true {
				t.Fatalf("Temp path not known: [%s]", relPath)
			}

			delete(tempPaths, relPath)

			return nil
		}

		filename := info.Name()
		relFilepath := path.Join(relParentPath, filename)

		// fmt.Printf("File> %s\n", relFilepath)

		j := tempFiles.Search(relFilepath)
		if j >= len(tempFiles) || tempFiles[j] != relFilepath {
			t.Fatalf("Handled file was not in the temporary-files list: [%s]", relFilepath)
			return nil
		}

		tempFiles = append(tempFiles[:j], tempFiles[j+1:]...)

		return nil
	}

	walk := NewWalk(tempPath, walkFunc)

	err := walk.Run()
	log.PanicIf(err)

	if rootNodeHit != true {
		t.Fatalf("Root node was never visited.")
	} else if len(tempPaths) != 0 {
		t.Fatalf("We expected one last directory (the root node): (%d)", len(tempPaths))
	} else if len(tempFiles) != 0 {
		t.Fatalf("Not all files were handled: %v", tempFiles)
	} else if walk.HasFinished() != true {
		t.Fatalf("HasFinished() is not true.")
	}
}

func ExampleWalk_Run() {
	// Stage test directory.

	fileCount := 20
	tempPath, tempFilenames := FillFlatTempPath(fileCount, nil)

	tempFilenames = append(tempFilenames, path.Base(tempPath))
	tempFilenames.Sort()

	// Walk

	defer func() {
		os.RemoveAll(tempPath)
	}()

	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		// Do your business.

		return nil
	}

	walk := NewWalk(tempPath, walkFunc)

	err := walk.Run()
	log.PanicIf(err)

	// Output:
}

func TestWalk_SetConcurrency(t *testing.T) {
	walk := new(Walk)
	walk.SetConcurrency(99)

	if walk.concurrency != 99 {
		t.Fatalf("'concurrency' field not correct: (%d)", walk.concurrency)
	}
}

func TestWalk_SetBufferSize(t *testing.T) {
	walk := new(Walk)
	walk.SetBufferSize(99)

	if walk.bufferSize != 99 {
		t.Fatalf("'bufferSize' field not correct: (%d)", walk.bufferSize)
	}
}

func TestNewWalk(t *testing.T) {
	flag := false
	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		flag = true

		return nil
	}

	w := NewWalk("root/path", walkFunc)
	if w.rootPath != "root/path" {
		t.Fatalf("rootPath not correct: [%s]", w.rootPath)
	} else if w.concurrency != defaultConcurrency {
		t.Fatalf("'concurrency' field not correct: (%d)", w.concurrency)
	} else if w.bufferSize != defaultBufferSize {
		t.Fatalf("'bufferSize' field not correct: (%d)", w.bufferSize)
	}

	w.walkFunc("", nil)
	if flag != true {
		t.Fatalf("walkFunc not correct.")
	}
}

func TestWalk_InitSync(t *testing.T) {
	walk := new(Walk)
	walk.InitSync()

	if walk.jobsC == nil {
		t.Fatalf("`jobsC` field not correct.")
	}

	close(walk.jobsC)

	if walk.wg == nil {
		t.Fatalf("`wg` field not correct.")
	}
}

func TestWalk_handleJobFileNode(t *testing.T) {
	sfi := rifs.NewSimpleFileInfoWithDirectory("file/name", time.Time{})

	hit := false
	walkFunc := func(parentPath string, info os.FileInfo) (err error) {
		if parentPath != "parent/path" {
			t.Fatalf("parentPath not correct: [%s]", parentPath)
		}

		if info.(*rifs.SimpleFileInfo) != sfi {
			t.Fatalf("FileInfo value not correct: [%s]", info.Name())
		}

		hit = true

		return nil
	}

	jfn := newJobFileNode("parent/path", sfi)

	walk := NewWalk("root/path", walkFunc)

	err := walk.handleJobFileNode(jfn)
	log.PanicIf(err)

	if hit != true {
		t.Fatalf("Callback not called.")
	}
}

func TestWalk_Stats(t *testing.T) {
	walk := new(Walk)

	walk.stats.DirectoriesVisited = 123

	stats := walk.Stats()
	if stats != walk.stats {
		t.Fatalf("Stats() does not return the right information.")
	}
}

func TestWalk_Stop(t *testing.T) {
	defer func() {
		err := recover().(error)
		if err.Error() != "close of closed channel" {
			log.Panic(err)
		}
	}()

	walk := NewWalk("root/path", nil)
	walk.InitSync()

	walk.Stop()

	// This should panic and short-circuit to the defer above.
	close(walk.jobsC)

	t.Fatalf("Expected close() call to fail. It should have been redundant.")
}

func TestWalk_HasFinished(t *testing.T) {
	w := Walk{}

	if w.HasFinished() != false {
		t.Fatalf("Initial HasFinished() wasn't false.")
	}

	w.hasFinished = true

	if w.HasFinished() != true {
		t.Fatalf("Final HasFinished() wasn't true.")
	}
}

func TestWalk_SetFilter__empty(t *testing.T) {
	walk := new(Walk)

	f := Filter{}
	walk.SetFilter(f)

	expectedFilter := internalFilter{
		includePaths:     make([]glob.Glob, 0),
		excludePaths:     make([]glob.Glob, 0),
		includeFilenames: make(sort.StringSlice, 0),
		excludeFilenames: make(sort.StringSlice, 0),
	}

	if reflect.DeepEqual(walk.filter, expectedFilter) != true {
		t.Fatalf("Filter not correct: %v", walk.filter)
	}
}

func TestWalk_SetFilter__nonempty(t *testing.T) {
	walk := new(Walk)

	f := Filter{
		IncludeFilenames: []string{"filename2", "filename1"},
		ExcludeFilenames: []string{"filename3", "filename4"},
	}

	walk.SetFilter(f)

	expectedFilter := internalFilter{
		includePaths:     make([]glob.Glob, 0),
		excludePaths:     make([]glob.Glob, 0),
		includeFilenames: sort.StringSlice{"filename1", "filename2"},
		excludeFilenames: sort.StringSlice{"filename3", "filename4"},
	}

	if reflect.DeepEqual(walk.filter, expectedFilter) != true {
		t.Fatalf("Filter not correct: %v", walk.filter)
	}
}
