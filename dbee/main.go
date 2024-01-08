package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/neovim/go-client/nvim"

	"github.com/kndndrj/nvim-dbee/dbee/handler"
	"github.com/kndndrj/nvim-dbee/dbee/plugin"
)

func main() {
	generateManifest := flag.String("manifest", "", "Generate manifest to file (filename of manifest).")
	getVersion := flag.Bool("version", false, "Get version and exit.")
	flag.Parse()

	// get version info
	if *getVersion {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			fmt.Println("unknown")
			os.Exit(1)
		}
		for _, inf := range info.Settings {
			if inf.Key == "vcs.revision" {
				fmt.Println(inf.Value)
				return
			}
		}
		fmt.Println("unknown")
		os.Exit(1)
	}

	stdout := os.Stdout
	os.Stdout = os.Stderr
	log.SetFlags(0)

	v, err := nvim.New(os.Stdin, stdout, stdout, log.Printf)
	if err != nil {
		log.Fatal(err)
	}

	logger := plugin.NewLogger(v)

	p := plugin.New(v, logger)

	h := handler.New(v, logger)
	defer h.Close()

	// configure "endpoints" from handler
	mountEndpoints(p, h)

	// generate manifest
	if *generateManifest != "" {
		err := p.Manifest("nvim_dbee", "dbee", *generateManifest)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("generated manifest to " + *generateManifest)
		return
	}

	// start server
	if err := v.Serve(); err != nil {
		log.Fatal(err)
	}
}
