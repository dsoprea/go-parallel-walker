package pathwalk

import (
	"os"
)

// Job describes any job being queued in the channel.
type Job interface {
	ParentNodePath() string
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

type jobDirectoryContentsBatch struct {
	parentPath string
	childBatch []string
}

func newJobDirectoryContentsBatch(parentPath string, childBatch []string) jobDirectoryContentsBatch {
	return jobDirectoryContentsBatch{
		parentPath: parentPath,
		childBatch: childBatch,
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
