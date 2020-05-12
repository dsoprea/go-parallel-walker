package pwtesting

import (
	"fmt"
	"os"
	"path"
	"sort"

	"io/ioutil"
	"math/rand"

	"github.com/dsoprea/go-logging"
	"github.com/google/uuid"
)

// FillFlatTempPath creates a temporary directory and fills a single
// subdirectory with test files.
func FillFlatTempPath(fileCount int, pathPrefix []string) (tempPath string, tempFilenames sort.StringSlice) {
	tempPath, err := ioutil.TempDir("", "")
	log.PanicIf(err)

	var effectiveTempPath string
	if pathPrefix != nil {
		suffixString := path.Join(pathPrefix...)
		effectiveTempPath = path.Join(tempPath, suffixString)

		err = os.MkdirAll(effectiveTempPath, 0755)
		log.PanicIf(err)
	} else {
		effectiveTempPath = tempPath
	}

	tempFilenames = make(sort.StringSlice, 0)
	for i := 0; i < fileCount; i++ {
		filename := fmt.Sprintf("temp-%d", i)
		filePath := path.Join(effectiveTempPath, filename)

		err := ioutil.WriteFile(filePath, []byte{}, 0)
		log.PanicIf(err)

		tempFilenames = append(tempFilenames, filename)
	}

	tempFilenames.Sort()

	return tempPath, tempFilenames
}

// FillHeirarchicalTempPath creates a temporary directory and filles a bunch of
// random-depth subdirectories with test-files.
func FillHeirarchicalTempPath(fileCount int, pathPrefix []string) (tempPath string, tempFiles sort.StringSlice) {
	tempPath, err := ioutil.TempDir("", "")
	log.PanicIf(err)

	var effectiveTempPath string
	if pathPrefix != nil {
		suffixString := path.Join(pathPrefix...)
		effectiveTempPath = path.Join(tempPath, suffixString)
	} else {
		effectiveTempPath = tempPath
	}

	tempFiles = make(sort.StringSlice, 0)
	for i := 0; i < fileCount; i++ {
		subdirectories := make([]string, 0)
		j := rand.Intn(3)
		for ; j >= 0; j-- {
			uuidPhrase := uuid.New().String()
			subdirectories = append(subdirectories, uuidPhrase)
		}

		subdirectoriesPhrase := path.Join(subdirectories...)
		tempPath := path.Join(effectiveTempPath, subdirectoriesPhrase)

		err = os.MkdirAll(tempPath, 0755)
		log.PanicIf(err)

		filename := fmt.Sprintf("temp-%d", i)
		filepath := path.Join(tempPath, filename)

		err := ioutil.WriteFile(filepath, []byte{}, 0)
		log.PanicIf(err)

		relFilepath := path.Join(subdirectoriesPhrase, filename)
		tempFiles = append(tempFiles, relFilepath)
	}

	tempFiles.Sort()

	return tempPath, tempFiles
}
