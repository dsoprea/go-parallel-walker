package pathwalk

import (
	"os"
)

// TODO(dustin): !! Add documentation.

type Job interface {
	ParentNodePath() string
}

type jobNode struct {
	parentNodePath string
}

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

func (jdn jobDirectoryNode) IsRootPath() bool {
	return jdn.jobNode.parentNodePath == ""
}

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

func (jdcb jobDirectoryContentsBatch) ParentNodePath() string {
	return jdcb.parentPath
}

func (jdcb jobDirectoryContentsBatch) ChildBatch() []string {
	return jdcb.childBatch
}
