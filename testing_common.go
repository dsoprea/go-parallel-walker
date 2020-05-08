package pathwalk

import (
	"fmt"
	"os"
	"path"
	"sort"

	"io/ioutil"

	"github.com/dsoprea/go-logging"
)

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
