package pathwalk

import (
	"fmt"
	"os"
)

// job describes any job being queued in the channel.
type job interface {
	ParentNodePath() string
	String() string
}

// jobNode is the default promoted type of our file and directory jbos.
type jobNode struct {
	parentNodePath string
}

// ParentNodePath is the full-path of the parent node.
func (jn jobNode) ParentNodePath() string {
	return jn.parentNodePath
}

type jobFileNode struct {
	jobNode
	info os.FileInfo
}

func newJobFileNode(parentNodePath string, info os.FileInfo) jobFileNode {
	jn := jobNode{
		parentNodePath: parentNodePath,
	}

	return jobFileNode{
		jobNode: jn,
		info:    info,
	}
}

// Info returns a `os.FileInfo`-compatible struct.
func (jfn jobFileNode) Info() os.FileInfo {
	return jfn.info
}

// String returns a descriptive string.
func (jfn jobFileNode) String() string {
	return fmt.Sprintf("JobFileNode<PARENT=[%s] NAME=[%s]>", jfn.jobNode.parentNodePath, jfn.info.Name())
}

type jobDirectoryNode struct {
	jobNode
	info os.FileInfo
}

func newJobDirectoryNode(parentNodePath string, info os.FileInfo) jobDirectoryNode {
	jn := jobNode{
		parentNodePath: parentNodePath,
	}

	return jobDirectoryNode{
		jobNode: jn,
		info:    info,
	}
}

// Info returns a `os.FileInfo`-compatible struct.
func (jdn jobDirectoryNode) Info() os.FileInfo {
	return jdn.info
}

// String returns a descriptive string.
func (jdn jobDirectoryNode) String() string {
	return fmt.Sprintf("JobDirectoryNode<PARENT=[%s] NAME=[%s]>", jdn.jobNode.parentNodePath, jdn.info.Name())
}

type jobDirectoryContentsBatch struct {
	parentPath     string
	batchNumber    int
	childBatch     []string
	doProcessFiles bool
}

func newJobDirectoryContentsBatch(parentPath string, batchNumber int, childBatch []string, doProcessFiles bool) jobDirectoryContentsBatch {
	return jobDirectoryContentsBatch{
		parentPath:     parentPath,
		batchNumber:    batchNumber,
		childBatch:     childBatch,
		doProcessFiles: doProcessFiles,
	}
}

// ParentNodePath is the full-path of the parent node.
func (jdcb jobDirectoryContentsBatch) ParentNodePath() string {
	return jdcb.parentPath
}

// ChildBatch is a string-slice of the entries in this batch.
func (jdcb jobDirectoryContentsBatch) ChildBatch() []string {
	return jdcb.childBatch
}

// String returns a descriptive string.
func (jdcb jobDirectoryContentsBatch) String() string {
	return fmt.Sprintf(
		"JobDirectoryContentsBatch<PARENT=[%s] BATCH=(%d) CHILD-COUNT=(%d)>",
		jdcb.parentPath, jdcb.batchNumber, len(jdcb.childBatch))
}

// DoProcessFiles returns whether the files should be sent to the callback. This
// is determined by the filters.
func (jdcb jobDirectoryContentsBatch) DoProcessFiles() bool {
	return jdcb.doProcessFiles
}
