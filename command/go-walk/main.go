package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"encoding/json"

	"github.com/dsoprea/go-logging"
	"github.com/jessevdk/go-flags"

	"github.com/dsoprea/go-parallel-walker"
)

type Parameters struct {
	RootPath string `short:"p" long:"root-path" required:"true" description:"Path to walk. This path will be included in the results."`

	ConcurrencyLevel int `short:"j" long:"concurrency" description:"Non-default maximum number of workers."`
	JobQueueLength   int `short:"q" long:"queue-length" description:"Non-default job-queue length."`

	IncludeFilenames  []string `short:"i" long:"include-filename" description:"Zero or more filename-patterns to include."`
	ExcludeFilenames  []string `short:"e" long:"exclude-filename" description:"Zero or more filename-patterns to exclude."`
	IncludePaths      []string `short:"I" long:"include-path" description:"Zero or more path-patterns to include. Use '**' for relative or recursive matching."`
	ExcludePaths      []string `short:"E" long:"exclude-path" description:"Zero or more path-patterns to exclude. Use '**' for relative or recursive matching."`
	IsCaseInsensitive bool     `short:"c" long:"case-insensitive" description:"Use case-insensitive matching"`

	DoJustPrintFiles       bool `short:"f" long:"just-files" description:"Just print files"`
	DoJustPrintDirectories bool `short:"d" long:"just-directories" description:"Just print directories"`
	DoPrintAsJson          bool `short:"J" long:"json" description:"Print as JSON"`
	DoPrintTypes           bool `short:"t" long:"type" description:"Prefix lines with entry types. Ignored if printing JSON."`

	DoPrintStats     bool `short:"s" long:"stats" description:"Print statistics. Ignored if printing JSON."`
	DoPrintVerbosity bool `short:"v" long:"verbose" description:"Print logging verbosity."`
}

var (
	arguments = new(Parameters)
)

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

	rootPath := strings.TrimRight(arguments.RootPath, "/")
	rootPathLen := len(rootPath) + 1

	m := sync.Mutex{}
	collected := make([]map[string]interface{}, 0)
	walkFunc := func(parentNodePath string, info os.FileInfo) (err error) {
		if arguments.DoJustPrintDirectories == true && info.IsDir() == false ||
			arguments.DoJustPrintFiles == true && info.IsDir() == true {
			return nil
		}

		fqName := path.Join(parentNodePath, info.Name())

		if fqName == rootPath {
			return nil
		}

		relName := fqName[rootPathLen:]

		if arguments.DoPrintAsJson == true {
			flat := map[string]interface{}{
				"path":         relName,
				"is_directory": info.IsDir(),
			}

			collected = append(collected, flat)
			return nil
		}

		m.Lock()

		if arguments.DoPrintTypes == true {
			var typeInitial string
			if info.IsDir() == true {
				typeInitial = "d"
			} else {
				typeInitial = "f"
			}

			fmt.Printf("%s ", typeInitial)
		}

		fmt.Printf("%s\n", relName)
		m.Unlock()

		return nil
	}

	walk := pathwalk.NewWalk(rootPath, walkFunc)

	if arguments.ConcurrencyLevel != 0 {
		walk.SetConcurrency(arguments.ConcurrencyLevel)
	}

	if arguments.JobQueueLength != 0 {
		walk.SetBufferSize(arguments.JobQueueLength)
	}

	filters := pathwalk.Filters{
		IncludePaths:     arguments.IncludePaths,
		ExcludePaths:     arguments.ExcludePaths,
		IncludeFilenames: arguments.IncludeFilenames,
		ExcludeFilenames: arguments.ExcludeFilenames,

		IsCaseInsensitive: arguments.IsCaseInsensitive,
	}

	walk.SetFilters(filters)

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
