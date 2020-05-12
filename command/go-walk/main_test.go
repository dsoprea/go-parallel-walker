package main

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"io/ioutil"

	"github.com/dsoprea/go-logging"
	"github.com/dsoprea/go-utility/testing"

	"github.com/dsoprea/go-parallel-walker"
)

func TestMain(t *testing.T) {
	ritesting.RedirectTty()

	defer func() {
		if errRaw := recover(); errRaw != nil {
			ritesting.RestoreAndDumpTty()
			fmt.Printf("Error!\n")

			err := errRaw.(error)
			log.PrintError(err)

			log.Panic(err)
		}

		ritesting.RestoreTty()
	}()

	originalArgs := os.Args

	defer func() {
		os.Args = originalArgs
	}()

	fileCount := 200
	tempPath, tempFilenames := pathwalk.FillFlatTempPath(fileCount, nil)

	tempFilenames.Sort()

	os.Args = []string{
		os.Args[0],
		tempPath,
	}

	main()

	os.Stdout.Close()

	raw, err := ioutil.ReadAll(ritesting.StdoutReader())
	log.PanicIf(err)

	ritesting.RestoreTty()

	output := string(raw)
	output = strings.TrimSpace(output)

	actual := strings.Split(output, "\n")

	actualSs := sort.StringSlice(actual)
	actualSs.Sort()

	if reflect.DeepEqual(actualSs, tempFilenames) != true {
		fmt.Printf("\n")
		fmt.Printf("Actual output:\n")
		fmt.Printf("\n")

		for i, line := range actualSs {
			fmt.Printf("ACTUAL> (%03d) [%s]\n", i, line)
		}

		fmt.Printf("\n")
		fmt.Printf("Expected output:\n")

		for i, tempFilename := range tempFilenames {
			fmt.Printf("EXPECTED> (%03d) [%s]\n", i, tempFilename)
		}

		fmt.Printf("\n")

		t.Fatalf("Filenames not correct/complete.")
	}
}
