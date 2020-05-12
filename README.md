[![Build Status](https://travis-ci.org/dsoprea/go-parallel-walker.svg?branch=master)](https://travis-ci.org/dsoprea/go-parallel-walker)
[![Coverage Status](https://coveralls.io/repos/github/dsoprea/go-parallel-walker/badge.svg?branch=master)](https://coveralls.io/github/dsoprea/go-parallel-walker?branch=master)
[![GoDoc](https://godoc.org/github.com/dsoprea/go-parallel-walker?status.svg)](https://godoc.org/github.com/dsoprea/go-parallel-walker)
[![Go Report Card](https://goreportcard.com/badge/github.com/dsoprea/go-parallel-walker)](https://goreportcard.com/report/github.com/dsoprea/go-parallel-walker)

# Overview

This package walks and processes a filesystem tree in parallel. A CLI frontend
is also provided.


# Features

- Can set non-default values for the worker-count, queue-length, and batch-
size parameters (for technical nit-pickers).
- Stat errors on directories and files will be ignored.
- Output can be formatted as JSON.
- Non-JSON output lines can include a file-type prefix.
- Both filename/extension- and directory-based filters are supported.
- Filtering supports both include- and exclude.
- Directory-based filters support `**` for recursive matching.
- Filters support case-insensitivity.
- There is full reporting with performance and directory metrics.
- MIME types can be detected and included in the output (just in the CLI, for convenience).
- Verbosity can be enabled to provide insight into include/exclude-related
  disqualifications.


# Library Support

For source-code documentation and examples, see the GoDoc badge/link above.


# Command-Line Support

Default output format:

```
$ go run command/go-walk/main.go ~/Downloads/nlp
20news-19997.tar.gz
ICPSR_34802-V1.zip
gdelt_20191018051500
trainingandtestdata.zip
blogs.zip
gdelt_20191018051500/20191018051500.gkg.csv.zip
gdelt_20191018051500/20191018051500.mentions.CSV.zip
gdelt_20191018051500/20191018051500.gkg.csv
gdelt_20191018051500/20191018051500.mentions.CSV
gdelt_20191018051500/20191018051500.export.CSV
gdelt_20191018051500/20191018051500.export.CSV.zip
```

With types:

```
$ go run command/go-walk/main.go ~/Downloads/nlp --type
f 20news-19997.tar.gz
f ICPSR_34802-V1.zip
d gdelt_20191018051500
f trainingandtestdata.zip
f blogs.zip
f gdelt_20191018051500/20191018051500.mentions.CSV
f gdelt_20191018051500/20191018051500.export.CSV
f gdelt_20191018051500/20191018051500.gkg.csv.zip
f gdelt_20191018051500/20191018051500.export.CSV.zip
f gdelt_20191018051500/20191018051500.mentions.CSV.zip
f gdelt_20191018051500/20191018051500.gkg.csv
```

With mime-types:

```
$ go run command/go-walk/main.go ~/Downloads/nlp --type --mime-type
d - gdelt_20191018051500
f application/zip ICPSR_34802-V1.zip
f application/zip trainingandtestdata.zip
f application/x-gzip 20news-19997.tar.gz
f application/zip gdelt_20191018051500/20191018051500.export.CSV.zip
f application/zip gdelt_20191018051500/20191018051500.mentions.CSV.zip
f text/plain; charset=utf-8 gdelt_20191018051500/20191018051500.gkg.csv
f text/plain; charset=utf-8 gdelt_20191018051500/20191018051500.mentions.CSV
f application/zip gdelt_20191018051500/20191018051500.gkg.csv.zip
f application/zip blogs.zip
f text/plain; charset=utf-8 gdelt_20191018051500/20191018051500.export.CSV
```

As JSON:

```
$ go run command/go-walk/main.go ~/Downloads/nlp --type --mime-type --json
[
    {
        "is_directory": false,
        "mime_type": "application/x-gzip",
        "mode": 420,
        "modified_time": "2019-10-17T02:05:47-04:00",
        "path": "20news-19997.tar.gz",
        "size": 17332201
    },
    {
        "is_directory": false,
        "mime_type": "application/zip",
        "mode": 420,
        "modified_time": "2019-10-17T02:19:03-04:00",
        "path": "trainingandtestdata.zip",
        "size": 81363704
    },
    {
        "is_directory": false,
        "mime_type": "application/zip",
        "mode": 420,
        "modified_time": "2019-10-17T01:58:02-04:00",
        "path": "blogs.zip",
        "size": 312949121
    },
    {
        "is_directory": false,
...
```

Just directories:

```
$ go run command/go-walk/main.go ~/Downloads/nlp --just-directories
gdelt_20191018051500
```

Just include the one subdirectory (with verbosity):

```
$ go run command/go-walk/main.go ~/Downloads/nlp --include-path 'gdelt_20191018051500' --verbose
2020/05/12 04:08:19 pathwalk.walk: [DEBUG]  Directory excluded: []
gdelt_20191018051500
gdelt_20191018051500/20191018051500.export.CSV
gdelt_20191018051500/20191018051500.export.CSV.zip
gdelt_20191018051500/20191018051500.mentions.CSV.zip
gdelt_20191018051500/20191018051500.gkg.csv
gdelt_20191018051500/20191018051500.gkg.csv.zip
gdelt_20191018051500/20191018051500.mentions.CSV
```

Exclude all ZIP-files (with verbosity):

```
$ go run command/go-walk/main.go ~/Downloads/nlp --exclude-filename '*.zip' --verbose
20news-19997.tar.gz
2020/05/12 04:09:34 pathwalk.walk: [DEBUG]  File excluded: [ICPSR_34802-V1.zip]
2020/05/12 04:09:34 pathwalk.walk: [DEBUG]  File excluded: [trainingandtestdata.zip]
2020/05/12 04:09:34 pathwalk.walk: [DEBUG]  File excluded: [blogs.zip]
gdelt_20191018051500
2020/05/12 04:09:34 pathwalk.walk: [DEBUG]  File excluded: [20191018051500.export.CSV.zip]
2020/05/12 04:09:34 pathwalk.walk: [DEBUG]  File excluded: [20191018051500.mentions.CSV.zip]
2020/05/12 04:09:34 pathwalk.walk: [DEBUG]  File excluded: [20191018051500.gkg.csv.zip]
gdelt_20191018051500/20191018051500.mentions.CSV
gdelt_20191018051500/20191018051500.gkg.csv
gdelt_20191018051500/20191018051500.export.CSV
```

Show statistics:

```
$ time go run command/go-walk/main.go ~/Pictures --stats >/dev/null
Processing Statistics
=====================
JobsDispatchedToNewWorker: (400)
JobsDispatchedToIdleWorker: (1001)
FilesVisited: (1361)
DirectoriesVisited: (15)
EntryBatchesProcessed: (25)
IdleWorkerTime: (2.180) seconds
DirectoriesIgnored: (0)
PathFilterIncludes: (0)
PathFilterExcludes: (0)
FileFilterIncludes: (0)
FileFilterExcludes: (0)



real    0m1.553s
user    0m0.852s
sys 0m0.277s


$ time go run command/go-walk/main.go ~/Downloads --stats >/dev/null
Processing Statistics
=====================
JobsDispatchedToNewWorker: (400)
JobsDispatchedToIdleWorker: (33560)
FilesVisited: (31014)
DirectoriesVisited: (1361)
EntryBatchesProcessed: (1585)
IdleWorkerTime: (172.312) seconds
DirectoriesIgnored: (0)
PathFilterIncludes: (0)
PathFilterExcludes: (0)
FileFilterIncludes: (0)
FileFilterExcludes: (0)



real    0m1.577s
user    0m1.434s
sys 0m0.629s



$ time go run command/go-walk/main.go ~/Downloads --stats --include-filename "*.jpg" >/dev/null
Processing Statistics
=====================
JobsDispatchedToNewWorker: (266)
JobsDispatchedToIdleWorker: (2955)
FilesVisited: (275)
DirectoriesVisited: (1361)
EntryBatchesProcessed: (1585)
IdleWorkerTime: (17.404) seconds
DirectoriesIgnored: (0)
PathFilterIncludes: (1361)
PathFilterExcludes: (0)
FileFilterIncludes: (275)
FileFilterExcludes: (30739)



real    0m1.561s
user    0m0.885s
sys 0m0.430s
```

The examples above use the "go run" method of calling the tool, but it is
obviously recommended to build the tool first and then call the binary.
