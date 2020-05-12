package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"encoding/json"

	"github.com/dsoprea/go-logging"
	"github.com/dsoprea/go-utility/data"
	"github.com/jessevdk/go-flags"

	"github.com/dsoprea/go-parallel-walker"
)

type parameters struct {
	Positional struct {
		RootPath string `positional-arg-name:"root_path" description:"Path to walk. This path will be included in the results."`
	} `positional-args:"yes" required:"yes"`

	ConcurrencyLevel int `short:"j" long:"concurrency" description:"Non-default maximum number of workers"`
	JobQueueLength   int `short:"q" long:"queue-length" description:"Non-default job-queue length"`
	BatchSize        int `short:"b" long:"batch-size" description:"Directory-processing batch-size"`

	IncludePaths      []string `short:"I" long:"include-path" description:"Zero or more path-patterns to include. Use '**' for relative or recursive matching."`
	ExcludePaths      []string `short:"E" long:"exclude-path" description:"Zero or more path-patterns to exclude. Use '**' for relative or recursive matching."`
	IncludeFilenames  []string `short:"i" long:"include-filename" description:"Zero or more filename-patterns to include"`
	ExcludeFilenames  []string `short:"e" long:"exclude-filename" description:"Zero or more filename-patterns to exclude"`
	IsCaseInsensitive bool     `short:"c" long:"case-insensitive" description:"Use case-insensitive matching"`

	DoJustPrintFiles       bool `short:"f" long:"just-files" description:"Just print files"`
	DoJustPrintDirectories bool `short:"d" long:"just-directories" description:"Just print directories"`
	DoPrintAsJson          bool `short:"J" long:"json" description:"Print as JSON"`
	DoPrintTypes           bool `short:"t" long:"type" description:"Prefix lines with entry types. Ignored if printing JSON."`

	DoPrintStats     bool `short:"s" long:"stats" description:"Print statistics. Ignored if printing JSON."`
	DoPrintVerbosity bool `short:"v" long:"verbose" description:"Print logging verbosity"`

	DoIncludeMimeType bool `short:"m" long:"mime-type" description:"Include MIME-types in the output. Prints hyphen for directories or for files that could not be processed."`
}

var (
	arguments = new(parameters)
)

var (
	rootPath    string
	rootPathLen int
)

func visitorFunction(outputLocker *sync.Mutex, rootPath string, parentNodePath string, info os.FileInfo, collected *[]map[string]interface{}) (err error) {
	if arguments.DoJustPrintDirectories == true && info.IsDir() == false ||
		arguments.DoJustPrintFiles == true && info.IsDir() == true {
		return nil
	}

	fqName := path.Join(parentNodePath, info.Name())

	if fqName == rootPath {
		return nil
	}

	relName := fqName[rootPathLen:]

	var mimeType string
	if info.IsDir() == false && arguments.DoIncludeMimeType == true {
		f, err := os.Open(fqName)
		if err == nil {
			mimeType, _ = ridata.DetectMimetype(f)
			f.Close()
		}
	}

	outputLocker.Lock()
	defer outputLocker.Unlock()

	if arguments.DoPrintAsJson == true {
		flat := map[string]interface{}{
			"path":          relName,
			"is_directory":  info.IsDir(),
			"size":          info.Size(),
			"modified_time": info.ModTime().Format(time.RFC3339),
			"mode":          info.Mode(),
		}

		if mimeType != "" {
			flat["mime_type"] = mimeType
		}

		collectedUpdated := append(*collected, flat)
		*collected = collectedUpdated

		return nil
	}

	if arguments.DoPrintTypes == true {
		var typeInitial string
		if info.IsDir() == true {
			typeInitial = "d"
		} else {
			typeInitial = "f"
		}

		fmt.Printf("%s ", typeInitial)
	}

	if arguments.DoIncludeMimeType == true {
		if mimeType != "" {
			fmt.Printf("%s ", mimeType)
		} else {
			fmt.Printf("- ")
		}
	}

	fmt.Printf("%s\n", relName)

	return nil
}

func main() {
	defer func() {
		if state := recover(); state != nil {
			err := log.Wrap(state.(error))
			log.PrintError(err)
			os.Exit(1)
		}
	}()

	p := flags.NewParser(arguments, flags.Default)

	_, err := p.Parse()
	if err != nil {
		os.Exit(1)
	}

	if arguments.DoPrintVerbosity == true {
		cla := log.NewConsoleLogAdapter()
		log.AddAdapter("console", cla)

		scp := log.NewStaticConfigurationProvider()
		scp.SetLevelName(log.LevelNameDebug)

		log.LoadConfiguration(scp)
	}

	rootPath = strings.TrimRight(arguments.Positional.RootPath, "/")
	rootPathLen = len(rootPath) + 1

	collected := make([]map[string]interface{}, 0)
	outputLocker := sync.Mutex{}
	visitorFunctionWrapper := func(parentNodePath string, info os.FileInfo) (err error) {
		err = visitorFunction(&outputLocker, rootPath, parentNodePath, info, &collected)
		log.PanicIf(err)

		return nil
	}

	walk := pathwalk.NewWalk(rootPath, visitorFunctionWrapper)

	if arguments.ConcurrencyLevel != 0 {
		walk.SetConcurrency(arguments.ConcurrencyLevel)
	}

	if arguments.JobQueueLength != 0 {
		walk.SetBufferSize(arguments.JobQueueLength)
	}

	if arguments.BatchSize != 0 {
		walk.SetBatchSize(arguments.BatchSize)
	}

	filter := pathwalk.Filter{
		IncludePaths:     arguments.IncludePaths,
		ExcludePaths:     arguments.ExcludePaths,
		IncludeFilenames: arguments.IncludeFilenames,
		ExcludeFilenames: arguments.ExcludeFilenames,

		IsCaseInsensitive: arguments.IsCaseInsensitive,
	}

	walk.SetFilter(filter)

	err = walk.Run()
	log.PanicIf(err)

	if arguments.DoPrintAsJson == true {
		je := json.NewEncoder(os.Stdout)
		je.SetIndent("", "    ")

		err := je.Encode(collected)
		log.PanicIf(err)
	} else if arguments.DoPrintStats == true {
		fmt.Printf("\n")

		walk.Stats().Dump()

		fmt.Printf("\n")
	}
}
