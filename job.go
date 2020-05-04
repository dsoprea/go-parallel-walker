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
	nodeName       string
}

func (jfl JobFileLeaf) ParentNodePath() string {
	return jfl.parentNodePath
}

func (jfl JobFileLeaf) NodeName() string {
	return jfl.nodeName
}

type jobFileNode struct {
	jobNode
	info os.FileInfo
}

func newJobFileNode(parentNodePath, nodeName string, info os.FileInfo) jobFileNode {
	jn := jobNode{
		parentNodePath: parentNodePath,
		nodeName:       nodeName,
	}

	return jobFileNode{
		jobNode: jobNode,
		info:    info,
	}
}

func (jfn jobFileNode) Info() os.FileInfo {
	return jfn.info
}

type jobDirectoryNode struct {
	jobNode
}

func newJobDirectoryNode(parentNodePath, nodeName string, info os.FileInfo) jobDirectoryNode {
	jn := jobNode{
		parentNodePath: parentNodePath,
		nodeName:       nodeName,
	}

	return jobDirectoryNode{
		jobNode: jobNode,
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
