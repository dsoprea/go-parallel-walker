package pathwalk

import (
	"reflect"
	"sort"
	"testing"
)

func TestinternalFilter_IsFileIncluded__includeOnly__hitOnInclude(t *testing.T) {
	filter := Filter{
		IncludeFilenames: []string{"filename2"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsFileIncluded("filename2") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsFileIncluded__includeOnly__missOnInclude(t *testing.T) {
	filter := Filter{
		IncludeFilenames: []string{"filename2"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsFileIncluded("filenameOther") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestinternalFilter_IsFileIncluded__includeOnly__caseSensitive(t *testing.T) {
	filter := Filter{
		IncludeFilenames: []string{"filename2"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsFileIncluded("filename2") != true {
		t.Fatalf("Expected include.")
	}

	if internalFilter.IsFileIncluded("Filename2") != false {
		t.Fatalf("Expected exclude.")
	}

	internalFilter.isCaseInsensitive = true

	if internalFilter.IsFileIncluded("filename2") != true {
		t.Fatalf("Expected include.")
	}

	if internalFilter.IsFileIncluded("Filename2") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsFileIncluded__excludeOnly__hitOnExclude(t *testing.T) {
	filter := Filter{
		ExcludeFilenames: []string{"filename2"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsFileIncluded("filename2") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestinternalFilter_IsFileIncluded__excludeOnly__missOnExclude(t *testing.T) {
	filter := Filter{
		ExcludeFilenames: []string{"filename2"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsFileIncluded("filenameOther") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsFileIncluded__excludeOnly__caseSensitive(t *testing.T) {
	filter := Filter{
		ExcludeFilenames: []string{"filename2"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsFileIncluded("filename2") != false {
		t.Fatalf("Expected exclude.")
	}

	if internalFilter.IsFileIncluded("Filename2") != true {
		t.Fatalf("Expected include.")
	}

	internalFilter.isCaseInsensitive = true

	if internalFilter.IsFileIncluded("filename2") != false {
		t.Fatalf("Expected exclude.")
	}

	if internalFilter.IsFileIncluded("Filename2") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestinternalFilter_IsFileIncluded__includeAndExclude__includesCheckedBeforeExcludes__includeOnPrefixAndPreemptExclude(t *testing.T) {
	filter := Filter{
		IncludeFilenames: []string{"filename2"},
		ExcludeFilenames: []string{"filename2"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsFileIncluded("filename2") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsFileIncluded__includeAndExclude__includesCheckedBeforeExcludes__includeOnPrefixAndExcludeOnWhole(t *testing.T) {
	filter := Filter{
		IncludeFilenames: []string{"included_file*"},
		ExcludeFilenames: []string{"included_file_nevermind"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsFileIncluded("included_file") != true {
		t.Fatalf("Expected include (1).")
	}

	if internalFilter.IsFileIncluded("included_file_nevermind") != true {
		t.Fatalf("Expected include (2).")
	}
}

func TestinternalFilter_IsFileIncluded__includeAndExclude__includesCheckedBeforeExcludes__missOnBoth(t *testing.T) {
	filter := Filter{
		IncludeFilenames: []string{"filename2"},
		ExcludeFilenames: []string{"filename3"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsFileIncluded("filenameOther") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestinternalFilter_IsFileIncluded__none__default(t *testing.T) {
	filter := Filter{}
	internalFilter := newInternalFilter(filter)

	if internalFilter.IsFileIncluded("filename") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsFileIncluded__none__explicit(t *testing.T) {
	filter := Filter{}
	internalFilter := newInternalFilter(filter)

	if internalFilter.IsFileIncluded("filename") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsPathIncluded__includeOnly__hitOnInclude(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"aa/bb"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("aa/bb") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsPathIncluded__includeOnly__caseInsensitive(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"aa/bb"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("aa/bb") != true {
		t.Fatalf("Expected include.")
	}

	if internalFilter.IsPathIncluded("Aa/bb") != false {
		t.Fatalf("Expected exclude.")
	}

	internalFilter.isCaseInsensitive = true

	if internalFilter.IsPathIncluded("aa/bb") != true {
		t.Fatalf("Expected include.")
	}

	if internalFilter.IsPathIncluded("Aa/bb") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsPathIncluded__includeOnly__hitOnInclude__absolute__pattern__oneComponent(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"aa/*x/bb"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("aa/bb") != false {
		t.Fatalf("Expected exclude.")
	}

	if internalFilter.IsPathIncluded("aa/xx/bb") != true {
		t.Fatalf("Expected include.")
	}

	if internalFilter.IsPathIncluded("aa/yy/bb") != false {
		t.Fatalf("Expected exclude.")
	}

	if internalFilter.IsPathIncluded("aa/xx/yy/bb") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestinternalFilter_IsPathIncluded__includeOnly__hitOnInclude__absolute__pattern__multipleComponentsDontMatch(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"aa/*bb"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("aa/xx/yy/zzbb") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestinternalFilter_IsPathIncluded__includeOnly__hitOnInclude__absolute__pattern__doesntMatchWhole(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"aa*bb"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("aa/xx/yy/zzbb") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestinternalFilter_IsPathIncluded__includeOnly__hitOnInclude__absolute__recursive(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"aa/**/bb"},
	}

	internalFilter := newInternalFilter(filter)

	// Notice that the /**/ can also be a zero-length match.
	if internalFilter.IsPathIncluded("aa/bb") != true {
		t.Fatalf("Expected include.")
	}

	if internalFilter.IsPathIncluded("aa/cc") != false {
		t.Fatalf("Expected exclude.")
	}

	if internalFilter.IsPathIncluded("aa/xx/bb") != true {
		t.Fatalf("Expected include.")
	}

	// Notice that the wildcard operator will match only one directory.
	if internalFilter.IsPathIncluded("aa/xx/yy/bb") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsPathIncluded__includeOnly__hitOnInclude__relative__left(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"**/aa/bb"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("root/path/aa/bb") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsPathIncluded__includeOnly__hitOnInclude__relative__right(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"aa/bb/**"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("aa/bb/sub1") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsPathIncluded__includeOnly__hitOnInclude__relative__both(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"**/aa/bb/**"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("root/path/aa/bb/sub1") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsPathIncluded__includeOnly__missOnInclude(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"aa/bb"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("cc/dd") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestinternalFilter_IsPathIncluded__excludeOnly__hitOnExclude(t *testing.T) {
	filter := Filter{
		ExcludePaths: []string{"aa/bb"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("aa/bb") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestinternalFilter_IsPathIncluded__excludeOnly__missOnExclude(t *testing.T) {
	filter := Filter{
		ExcludePaths: []string{"aa/bb"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("cc/dd") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsPathIncluded__excludeOnly__caseInsensitive(t *testing.T) {
	filter := Filter{
		ExcludePaths: []string{"aa/bb"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("aa/bb") != false {
		t.Fatalf("Expected exclude.")
	}

	if internalFilter.IsPathIncluded("Aa/bb") != true {
		t.Fatalf("Expected include.")
	}

	internalFilter.isCaseInsensitive = true

	if internalFilter.IsPathIncluded("aa/bb") != false {
		t.Fatalf("Expected exclude.")
	}

	if internalFilter.IsPathIncluded("Aa/bb") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestinternalFilter_IsPathIncluded__includeAndExclude__includesCheckedBeforeExcludes__includeOnPrefixAndPreemptExclude(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"path1/path2"},
		ExcludePaths: []string{"path1/path2"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("path1/path2") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsPathIncluded__includeAndExclude__includesCheckedBeforeExcludes__includeOnPrefixAndExcludeOnWhole(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"path1/path2"},
		ExcludePaths: []string{"path1/path2/path3"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("path1/path2/path3") != false {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsPathIncluded__includeAndExclude__includesCheckedBeforeExcludes__missOnBoth(t *testing.T) {
	filter := Filter{
		IncludePaths: []string{"path1/path2"},
		ExcludePaths: []string{"path3/path4"},
	}

	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("path/other") != false {
		t.Fatalf("Expected exclude.")
	}
}

func TestinternalFilter_IsPathIncluded__none__default(t *testing.T) {
	filter := Filter{}
	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("some/path") != true {
		t.Fatalf("Expected include.")
	}
}

func TestinternalFilter_IsPathIncluded__none__explicit(t *testing.T) {
	filter := Filter{}
	internalFilter := newInternalFilter(filter)

	if internalFilter.IsPathIncluded("some/path") != true {
		t.Fatalf("Expected include.")
	}
}

func TestNewInternalFilters(t *testing.T) {
	f := Filter{
		IncludePaths:     []string{"aa/bb"},
		ExcludePaths:     []string{"cc/dd"},
		IncludeFilenames: []string{"filename2", "filename1"},
		ExcludeFilenames: []string{"filename3", "filename4"},
	}

	internal := newInternalFilter(f)

	expectedFilters := internalFilter{
		includePaths:     internal.includePaths,
		excludePaths:     internal.excludePaths,
		includeFilenames: sort.StringSlice{"filename1", "filename2"},
		excludeFilenames: sort.StringSlice{"filename3", "filename4"},
	}

	if reflect.DeepEqual(internal, expectedFilters) != true {
		t.Fatalf("Filters not correct: %v", internal)
	}
}
