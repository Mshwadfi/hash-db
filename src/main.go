package main

import (
	"fmt"
	"os"

	"mydb/src/catalog"
	"mydb/src/executor"
	"mydb/src/repl"
	"mydb/src/storageLayer"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	dm, err := storageLayer.OpenDiskManager("data.db")
	if err != nil {
		return err
	}
	defer dm.Close()

	bm, err := storageLayer.NewBufferManager(10, dm)
	if err != nil {
		return err
	}
	defer bm.FlushAll()

	cat, err := catalog.OpenCatalog(bm, dm)
	if err != nil {
		return err
	}

	return repl.NewRepl(executor.NewExecutor(cat), os.Stdin, os.Stdout).Run()
}
