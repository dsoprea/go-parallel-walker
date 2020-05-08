package pathwalk

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"sync"
	"testing"
	"time"

	"io/ioutil"

	"github.com/dsoprea/go-logging"
	"github.com/dsoprea/go-utility/filesystem"
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

	// Stop the test ASAP.
	close(jobsC)

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
	walk.initSync()

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
	walk.initSync()

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
	walk.initSync()

	// Handle the root directory node.

	sfi := rifs.NewSimpleFileInfoWithDirectory("testdir", time.Time{})
	jdn := newJobDirectoryNode(tempPath, sfi)

	// This will fork workers to process the children in batches.
	err := walk.handleJobDirectoryNode(jdn)
	log.PanicIf(err)

	// The workers will starve and shutdown almost immediately.
	walk.wg.Wait()
	close(walk.jobsC)

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

	tempPath, err := ioutil.TempDir("", "")
	log.PanicIf(err)

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
	walk.initSync()

	// Handle the root directory node.

	// We copy the slice going in as the argument to prevent races with the
	// modification in the callback above.
	childBatch := make([]string, len(tempFilenames))
	copy(childBatch, tempFilenames)

	jdcb := newJobDirectoryContentsBatch(tempPath, childBatch)

	// This will fork workers to process the children in batches.
	err = walk.handleJobDirectoryContentsBatch(jdcb)
	log.PanicIf(err)

	// The workers will starve and shutdown almost immediately.
	walk.wg.Wait()
	close(walk.jobsC)

	if len(tempFilenames) != 0 {
		t.Fatalf("Not all files were handled: %v", tempFilenames)
	}
}

func TestWalk_Run__simple(t *testing.T) {
	// Stage test directory.

	tempPath, err := ioutil.TempDir("", "")
	log.PanicIf(err)

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

	err = walk.Run()
	log.PanicIf(err)

	if len(tempFilenames) != 0 {
		t.Fatalf("Not all files were handled: %v", tempFilenames)
	}
}

func ExampleWalk_Run() {
	// Stage test directory.

	tempPath, err := ioutil.TempDir("", "")
	log.PanicIf(err)

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

	err = walk.Run()
	log.PanicIf(err)

	// Output:
}
