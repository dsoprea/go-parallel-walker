package pathwalk

import (
	"sort"
	"strings"

	"path/filepath"

	"github.com/dsoprea/go-logging"
	"github.com/gobwas/glob"
)

// Filters define the parameters that can be provided by the user to control the
// walk.
type Filters struct {
	IncludePaths     []string
	ExcludePaths     []string
	IncludeFilenames []string
	ExcludeFilenames []string

	IsCaseInsensitive bool
}

// internalFilters is a conditioned copy of the user filtering parameters.
type internalFilters struct {
	includePaths     []glob.Glob
	excludePaths     []glob.Glob
	includeFilenames sort.StringSlice
	excludeFilenames sort.StringSlice

	isCaseInsensitive bool
}

// IsFileIncluded determines if the given filename should be visited.
func (filters internalFilters) IsFileIncluded(filename string) bool {
	if filters.isCaseInsensitive == true {
		filename = strings.ToLower(filename)
	}

	if len(filters.includeFilenames) > 0 {
		// If any included-files are declared, then any unmatched files will be
		// skipped.

		for _, includePattern := range filters.includeFilenames {
			hit, err := filepath.Match(includePattern, filename)
			log.PanicIf(err)

			if hit == true {
				return true
			}
		}

		return false
	}

	if len(filters.excludeFilenames) > 0 {
		// If any included-files are declared, then any unmatched files will be
		// skipped.

		for _, excludePattern := range filters.excludeFilenames {
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
func (filters internalFilters) IsPathIncluded(currentPath string) bool {
	if filters.isCaseInsensitive == true {
		currentPath = strings.ToLower(currentPath)
	}

	if len(filters.includePaths) > 0 {
		// If any included-files are declared, then any unmatched files will be
		// skipped.

		for _, includePattern := range filters.includePaths {
			if includePattern.Match(currentPath) == true {
				return true
			}
		}

		return false
	}

	if len(filters.excludePaths) > 0 {
		// If any included-files are declared, then any unmatched files will be
		// skipped.

		for _, excludePattern := range filters.excludePaths {
			if excludePattern.Match(currentPath) == true {
				return false
			}
		}
	}

	// No include filters or matching exclude filters. Include.
	return true
}

// SetFilters sets filtering parameters for the next call to Run(). Behavior is
// undefined if this is changed *during* a call to `Run()`. The filters will be
// sorted automatically.
func newInternalFilters(filters Filters) internalFilters {

	internalFilters := internalFilters{
		isCaseInsensitive: filters.IsCaseInsensitive,
	}

	internalFilters.includePaths = make([]glob.Glob, 0)

	if filters.IncludePaths != nil {
		includePatterns := sort.StringSlice(filters.IncludePaths)
		sort.Sort(sort.Reverse(includePatterns))

		for _, includePattern := range includePatterns {
			internalFilters.includePaths = append(internalFilters.includePaths, glob.MustCompile(includePattern, '/'))
		}
	}

	internalFilters.excludePaths = make([]glob.Glob, 0)

	if filters.ExcludePaths != nil {
		excludePatterns := sort.StringSlice(filters.ExcludePaths)
		sort.Sort(sort.Reverse(excludePatterns))

		for _, excludePattern := range excludePatterns {
			internalFilters.excludePaths = append(internalFilters.excludePaths, glob.MustCompile(excludePattern, '/'))
		}
	}

	if filters.IncludeFilenames == nil {
		internalFilters.includeFilenames = make(sort.StringSlice, 0)
	} else {
		internalFilters.includeFilenames = sort.StringSlice(filters.IncludeFilenames)
		internalFilters.includeFilenames.Sort()
	}

	if filters.ExcludeFilenames == nil {
		internalFilters.excludeFilenames = make(sort.StringSlice, 0)
	} else {
		internalFilters.excludeFilenames = sort.StringSlice(filters.ExcludeFilenames)
		internalFilters.excludeFilenames.Sort()
	}

	return internalFilters
}
