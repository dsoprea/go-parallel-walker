package pathwalk

import (
	"testing"
)

func TestWalk_IsFileIncluded__includeOnly__hitOnInclude(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludeFilenames: []string{"filename2"},
	}

	walk.SetFilters(f)

	if walk.filters.IsFileIncluded("filename2") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsFileIncluded__includeOnly__missOnInclude(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludeFilenames: []string{"filename2"},
	}

	walk.SetFilters(f)

	if walk.filters.IsFileIncluded("filenameOther") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestWalk_IsFileIncluded__excludeOnly__hitOnExclude(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		ExcludeFilenames: []string{"filename2"},
	}

	walk.SetFilters(f)

	if walk.filters.IsFileIncluded("filename2") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestWalk_IsFileIncluded__excludeOnly__missOnExclude(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		ExcludeFilenames: []string{"filename2"},
	}

	walk.SetFilters(f)

	if walk.filters.IsFileIncluded("filenameOther") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsFileIncluded__includeAndExclude__includesCheckedBeforeExcludes__includeOnPrefixAndPreemptExclude(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludeFilenames: []string{"filename2"},
		ExcludeFilenames: []string{"filename2"},
	}

	walk.SetFilters(f)

	if walk.filters.IsFileIncluded("filename2") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsFileIncluded__includeAndExclude__includesCheckedBeforeExcludes__includeOnPrefixAndExcludeOnWhole(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludeFilenames: []string{"included_file*"},
		ExcludeFilenames: []string{"included_file_nevermind"},
	}

	walk.SetFilters(f)

	if walk.filters.IsFileIncluded("included_file") != true {
		t.Fatalf("Expected include (1).")
	}

	if walk.filters.IsFileIncluded("included_file_nevermind") != true {
		t.Fatalf("Expected include (2).")
	}
}

func TestWalk_IsFileIncluded__includeAndExclude__includesCheckedBeforeExcludes__missOnBoth(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludeFilenames: []string{"filename2"},
		ExcludeFilenames: []string{"filename3"},
	}

	walk.SetFilters(f)

	if walk.filters.IsFileIncluded("filenameOther") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestWalk_IsFileIncluded__none__default(t *testing.T) {
	walk := new(Walk)

	if walk.filters.IsFileIncluded("filename") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsFileIncluded__none__explicit(t *testing.T) {
	walk := new(Walk)

	f := Filters{}

	walk.SetFilters(f)

	if walk.filters.IsFileIncluded("filename") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsPathIncluded__includeOnly__hitOnInclude(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludePaths: []string{"aa/bb"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("aa/bb") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsPathIncluded__includeOnly__hitOnInclude__absolute__pattern__oneComponent(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludePaths: []string{"aa/*x/bb"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("aa/bb") != false {
		t.Fatalf("Expected exclude.")
	}

	if walk.filters.IsPathIncluded("aa/xx/bb") != true {
		t.Fatalf("Expected include.")
	}

	if walk.filters.IsPathIncluded("aa/yy/bb") != false {
		t.Fatalf("Expected exclude.")
	}

	if walk.filters.IsPathIncluded("aa/xx/yy/bb") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestWalk_IsPathIncluded__includeOnly__hitOnInclude__absolute__pattern__multipleComponentsDontMatch(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludePaths: []string{"aa/*bb"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("aa/xx/yy/zzbb") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestWalk_IsPathIncluded__includeOnly__hitOnInclude__absolute__pattern__doesntMatchWhole(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludePaths: []string{"aa*bb"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("aa/xx/yy/zzbb") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestWalk_IsPathIncluded__includeOnly__hitOnInclude__absolute__recursive(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludePaths: []string{"aa/**/bb"},
	}

	walk.SetFilters(f)

	// Notice that the /**/ can also be a zero-length match.
	if walk.filters.IsPathIncluded("aa/bb") != true {
		t.Fatalf("Expected include.")
	}

	if walk.filters.IsPathIncluded("aa/cc") != false {
		t.Fatalf("Expected exclude.")
	}

	if walk.filters.IsPathIncluded("aa/xx/bb") != true {
		t.Fatalf("Expected include.")
	}

	// Notice that the wildcard operator will match only one directory.
	if walk.filters.IsPathIncluded("aa/xx/yy/bb") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsPathIncluded__includeOnly__hitOnInclude__relative__left(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludePaths: []string{"**/aa/bb"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("root/path/aa/bb") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsPathIncluded__includeOnly__hitOnInclude__relative__right(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludePaths: []string{"aa/bb/**"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("aa/bb/sub1") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsPathIncluded__includeOnly__hitOnInclude__relative__both(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludePaths: []string{"**/aa/bb/**"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("root/path/aa/bb/sub1") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsPathIncluded__includeOnly__missOnInclude(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludePaths: []string{"aa/bb"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("cc/dd") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestWalk_IsPathIncluded__excludeOnly__hitOnExclude(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		ExcludePaths: []string{"aa/bb"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("aa/bb") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestWalk_IsPathIncluded__excludeOnly__missOnExclude(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		ExcludePaths: []string{"aa/bb"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("cc/dd") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsPathIncluded__includeAndExclude__includesCheckedBeforeExcludes__includeOnPrefixAndPreemptExclude(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludePaths: []string{"path1/path2"},
		ExcludePaths: []string{"path1/path2"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("path1/path2") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsPathIncluded__includeAndExclude__includesCheckedBeforeExcludes__includeOnPrefixAndExcludeOnWhole(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludePaths: []string{"path1/path2"},
		ExcludePaths: []string{"path1/path2/path3"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("path1/path2/path3") != false {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsPathIncluded__includeAndExclude__includesCheckedBeforeExcludes__missOnBoth(t *testing.T) {
	walk := new(Walk)

	f := Filters{
		IncludePaths: []string{"path1/path2"},
		ExcludePaths: []string{"path3/path4"},
	}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("path/other") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestWalk_IsPathIncluded__none__default(t *testing.T) {
	walk := new(Walk)

	if walk.filters.IsPathIncluded("some/path") != true {
		t.Fatalf("Expected include.")
	}
}

func TestWalk_IsPathIncluded__none__explicit(t *testing.T) {
	walk := new(Walk)

	f := Filters{}

	walk.SetFilters(f)

	if walk.filters.IsPathIncluded("some/path") != true {
		t.Fatalf("Expected include.")
	}
}
