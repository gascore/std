package main

import (
	"os"
	"github.com/gascore/gasx"
	"github.com/gascore/gasx/html"
)

/*
	components/builder compile *.gox files to *.go
*/

func main() {
	htmlCompiler := html.NewCompiler()
	builder := &gasx.Builder{
		BlockCompilers: []gasx.BlockCompiler{
			htmlCompiler.Block(),
		},
	}

	currentDir, err := os.Getwd()
	gasx.Must(err)

	files, err := gasx.GasFilesCustomDir(currentDir+"/../", []string{"gos"})
	gasx.Must(err)

	gasx.Must(builder.ParseFiles(files))

	gasx.Log("Builded")
}