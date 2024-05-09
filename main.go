package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunfmin/gogen"
	"github.com/sunfmin/snippetgo/parse"
)

var pkg = flag.String("pkg", "generated", "generated package name")
var dir = flag.String("dir", ".", "source code dir to scan examples")
var relBase = flag.String("rel-base", ".", "the path on which the file of the snippet result is based for filepath.Rel")
var sdirs = flag.String("skip-dir", "", "comma separate dirs to skip like '.git/,node_modules/'")

var skipDirs = []string{
	"node_modules/",
	".git/",
	"dist/",
}

func main() {
	flag.Parse()
	if len(*sdirs) > 0 {
		skipDirs = strings.Split(*sdirs, ",")
		for i, d := range skipDirs {
			skipDirs[i] = strings.TrimSpace(d)
		}
	}

	var err error
	var relBaseAbs string
	if *relBase != "" {
		relBaseAbs, err = filepath.Abs(*relBase)
		if err != nil {
			panic(err)
		}
	}

	gf := gogen.File("f.go").Package(*pkg)

	imported := false
	err = filepath.Walk(*dir, func(path string, f os.FileInfo, err error) error {

		for _, dir := range skipDirs {
			if strings.Index(path, dir) >= 0 {
				// fmt.Println("skipping dir", path)
				return filepath.SkipDir
			}
		}

		if f.IsDir() {
			// fmt.Println("is dir", path)
			return nil
		}

		// to support other source files like js, ts, json
		// if !strings.HasSuffix(f.Name(), ".go") {
		//	 return nil
		// }

		// fmt.Println("is file", path)
		snippets, err := parse.Snippets(path)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
		}

		if len(snippets) > 0 && !imported {
			gf.Body(
				gogen.Imports(
					"github.com/sunfmin/snippetgo/parse",
				),
			)
			imported = true
		}

		for _, s := range snippets {
			locationFile, err := filepath.Abs(s.Location.File)
			if err != nil {
				return err
			}
			if relBaseAbs != "" {
				relPath, err := filepath.Rel(relBaseAbs, locationFile)
				if err != nil {
					return fmt.Errorf("filepath.Rel err: %w", err)
				}
				locationFile = relPath
			}

			gf.BodySnippet(`
var $NAME = string($BYTE)
var $NAMELocation = parse.Location{File: $FILE, StartLine: $START, EndLine: $END}
`,
				"$NAME", s.Name,
				"$BYTE", fmt.Sprintf("%#+v", []byte(s.Code)),
				"$FILE", fmt.Sprintf("%q", locationFile),
				"$START", fmt.Sprint(s.Location.StartLine),
				"$END", fmt.Sprint(s.Location.EndLine),
			)
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	err = gf.Fprint(os.Stdout, context.TODO())
	if err != nil {
		panic(err)
	}
}
