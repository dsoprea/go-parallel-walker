package pathwalk

import (
	"reflect"
	"testing"
	"time"

	"github.com/dsoprea/go-utility/filesystem"
)

func TestJobNode_ParentNodePath(t *testing.T) {
	jn := jobNode{
		parentNodePath: "abc",
	}

	if jn.ParentNodePath() != "abc" {
		t.Fatalf("`ParentNodePath()` accessor does not return correct value: [%s]", jn.ParentNodePath())
	}
}

func TestNewJobFileNode(t *testing.T) {
	testFileInfo := rifs.NewSimpleFileInfoWithFile("test.file", 0, 0, time.Time{})
	jfn := newJobFileNode("parent/path", testFileInfo)

	if jfn.jobNode.ParentNodePath() != "parent/path" {
		t.Fatalf("`ParentNodePath()` accessor does not return correct value: [%s]", jfn.jobNode.ParentNodePath())
	} else if jfn.info != testFileInfo {
		t.Fatalf("`info` is not correct value: %v", jfn.info)
	}
}

func TestJobFileNode_Info(t *testing.T) {
	testFileInfo := rifs.NewSimpleFileInfoWithFile("test.file", 0, 0, time.Time{})
	jfn := jobFileNode{
		info: testFileInfo,
	}

	if jfn.Info() != testFileInfo {
		t.Fatalf("`Info()` accessor does not return correct value: %v", jfn.Info())
	}
}

func TestJobFileNode_String(t *testing.T) {
	testFileInfo := rifs.NewSimpleFileInfoWithFile("test.file", 0, 0, time.Time{})
	jfn := newJobFileNode("parent/path", testFileInfo)

	if jfn.String() != "JobFileNode<PARENT=[parent/path] NAME=[test.file]>" {
		t.Fatalf("`String()` accessor does not return correct value: %v", jfn.String())
	}
}

func TestNewJobDirectoryNode(t *testing.T) {
	testDirInfo := rifs.NewSimpleFileInfoWithDirectory("test/path/subdirectory", time.Time{})
	jdn := newJobDirectoryNode("test/path", testDirInfo)

	if jdn.jobNode.ParentNodePath() != "test/path" {
		t.Fatalf("ParentNodePath() accessor does not return correct value: [%s]", jdn.jobNode.ParentNodePath())
	} else if jdn.info != testDirInfo {
		t.Fatalf("`info` field does not have correct value: %v", jdn.info)
	}
}

func TestJobDirectoryNode_Info(t *testing.T) {
	testDirInfo := rifs.NewSimpleFileInfoWithDirectory("test/path", time.Time{})

	jdn := jobDirectoryNode{
		info: testDirInfo,
	}

	if jdn.Info() != testDirInfo {
		t.Fatalf("`Info()` accessor does not return correct value: %v", jdn.Info())
	}
}

func TestJobDirectoryNode_String(t *testing.T) {
	testDirInfo := rifs.NewSimpleFileInfoWithDirectory("test/path/some_sub", time.Time{})
	jdn := newJobDirectoryNode("test/path", testDirInfo)

	if jdn.String() != "JobDirectoryNode<PARENT=[test/path] NAME=[test/path/some_sub]>" {
		t.Fatalf("`String()` accessor does not return correct value: %v", jdn.String())
	}
}

func TestNewJobDirectoryContentsBatch(t *testing.T) {
	childBatch := []string{
		"zz",
		"aa",
	}

	jdcb := newJobDirectoryContentsBatch("parent/path", 22, childBatch, true)

	if jdcb.parentPath != "parent/path" {
		t.Fatalf("`parentNodePath` field does not have correct value: [%s]", jdcb.parentPath)
	} else if jdcb.batchNumber != 22 {
		t.Fatalf("`batchNumber` field does not have correct value: (%d)", jdcb.batchNumber)
	} else if reflect.DeepEqual(jdcb.childBatch, childBatch) != true {
		t.Fatalf("`childBatch` field does not have correct value: %v", jdcb.childBatch)
	} else if jdcb.doProcessFiles != true {
		t.Fatalf("`doProcessFiles` field does not have correct value: %v", jdcb.doProcessFiles)
	}
}

func TestJobDirectoryContentsBatch_ParentPath(t *testing.T) {
	jdcb := jobDirectoryContentsBatch{
		parentPath: "a/b/c",
	}

	if jdcb.ParentNodePath() != "a/b/c" {
		t.Fatalf("`ParentNodePath()` accessor did not return the correct value: [%s]", jdcb.ParentNodePath())
	}
}

func TestJobDirectoryContentsBatch_ChildBatch(t *testing.T) {
	childBatch := []string{
		"zz",
		"aa",
	}

	jdcb := jobDirectoryContentsBatch{
		childBatch: childBatch,
	}

	if reflect.DeepEqual(jdcb.ChildBatch(), childBatch) != true {
		t.Fatalf("`ChildBatch()` accessor does not return correct value: %v", jdcb.ChildBatch())
	}
}

func TestJobDirectoryContentsBatch_String(t *testing.T) {
	childBatch := []string{
		"zz",
		"aa",
	}

	jdcb := newJobDirectoryContentsBatch("parent/path", 22, childBatch, true)

	if reflect.DeepEqual(jdcb.childBatch, childBatch) != true {
		t.Fatalf("`childBatch` field does not have correct value: %v", jdcb.childBatch)
	}
}

func TestJobDirectoryContentsBatch_DoProcessFiles(t *testing.T) {
	jdcb := jobDirectoryContentsBatch{
		doProcessFiles: true,
	}

	if jdcb.DoProcessFiles() != true {
		t.Fatalf("`DoProcessFiles()` accessor did not return the correct value: [%v]", jdcb.DoProcessFiles())
	}
}
