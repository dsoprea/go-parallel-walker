package pathwalk

import (
	"sort"
	"strings"

	"path/filepath"

	"github.com/dsoprea/go-logging"
	"github.com/gobwas/glob"
)

// Filter define the parameters that can be provided by the user to control the
// walk.
type Filter struct {
	IncludePaths     []string
	ExcludePaths     []string
	IncludeFilenames []string
	ExcludeFilenames []string

	IsCaseInsensitive bool
}

// internalFilter is a conditioned copy of the user filtering parameters.
type internalFilter struct {
	includePaths     []glob.Glob
	excludePaths     []glob.Glob
	includeFilenames sort.StringSlice
	excludeFilenames sort.StringSlice

	isCaseInsensitive bool
}

// IsFileIncluded determines if the given filename should be visited.
func (filter internalFilter) IsFileIncluded(filename string) bool {
	if filter.isCaseInsensitive == true {
		filename = strings.ToLower(filename)
	}

	if len(filter.includeFilenames) > 0 {
		// If any included-files are declared, then any unmatched files will be
		// skipped.

		for _, includePattern := range filter.includeFilenames {
			hit, err := filepath.Match(includePattern, filename)
			log.PanicIf(err)

			if hit == true {
				return true
			}
		}

		return false
	}

	if len(filter.excludeFilenames) > 0 {
		// If any included-files are declared, then any unmatched files will be
		// skipped.

		for _, excludePattern := range filter.excludeFilenames {
			hit, err := filepath.Match(excludePattern, filename)
			log.PanicIf(err)

			if hit == true {
				return false
			}

		}
	}

	// No include filters or matching exclude filters. Include.
	return true
}

// IsPathIncluded determines if the given path should be visited.
func (filter internalFilter) IsPathIncluded(currentPath string) bool {
	if filter.isCaseInsensitive == true {
		currentPath = strings.ToLower(currentPath)
	}

	if len(filter.includePaths) > 0 {
		// If any included-files are declared, then any unmatched files will be
		// skipped.

		for _, includePattern := range filter.includePaths {
			if includePattern.Match(currentPath) == true {
				return true
			}
		}

		return false
	}

	if len(filter.excludePaths) > 0 {
		// If any included-files are declared, then any unmatched files will be
		// skipped.

		for _, excludePattern := range filter.excludePaths {
			if excludePattern.Match(currentPath) == true {
				return false
			}
		}
	}

	// No include filters or matching exclude filters. Include.
	return true
}

// newInternalFilters constructs an `internalFilter` from a `Filter`.
func newInternalFilter(filter Filter) internalFilter {

	internalFilter := internalFilter{
		isCaseInsensitive: filter.IsCaseInsensitive,
	}

	internalFilter.includePaths = make([]glob.Glob, 0)

	if filter.IncludePaths != nil {
		includePatterns := sort.StringSlice(filter.IncludePaths)
		sort.Sort(sort.Reverse(includePatterns))

		for _, includePattern := range includePatterns {
			internalFilter.includePaths = append(internalFilter.includePaths, glob.MustCompile(includePattern, '/'))
		}
	}

	internalFilter.excludePaths = make([]glob.Glob, 0)

	if filter.ExcludePaths != nil {
		excludePatterns := sort.StringSlice(filter.ExcludePaths)
		sort.Sort(sort.Reverse(excludePatterns))

		for _, excludePattern := range excludePatterns {
			internalFilter.excludePaths = append(internalFilter.excludePaths, glob.MustCompile(excludePattern, '/'))
		}
	}

	if filter.IncludeFilenames == nil {
		internalFilter.includeFilenames = make(sort.StringSlice, 0)
	} else {
		internalFilter.includeFilenames = sort.StringSlice(filter.IncludeFilenames)
		internalFilter.includeFilenames.Sort()
	}

	if filter.ExcludeFilenames == nil {
		internalFilter.excludeFilenames = make(sort.StringSlice, 0)
	} else {
		internalFilter.excludeFilenames = sort.StringSlice(filter.ExcludeFilenames)
		internalFilter.excludeFilenames.Sort()
	}

	return internalFilter
}
