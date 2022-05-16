package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/jawher/mow.cli"
)

func openCSV(filename string) (*csv.Reader, []string, error) {
	file, err := os.Open(filename)
    if err != nil {
		return nil, nil, err
    }
    reader := csv.NewReader(file)

	argnames, err := reader.Read()
	if err != nil {
		return nil, nil, err
	}

	return reader, argnames, nil
}

func runner(command string, parallelism int, csvfile string, fixedargs []string) error {

	argreader, argnames, err := openCSV(csvfile)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}

	distribution := make(chan []string, parallelism)

	wg := new(sync.WaitGroup)

	for i := 0; i < parallelism; i += 1 {
		wg.Add(1)
		go func() {
			for {
				args := <- distribution
				if len(args) == 0 {
					break
				}

				if len(args) != len(argnames) {
					log.Printf("Skipping run as %v and %v don't match", args, argnames)
					continue
				}

				fullarglist := make([]string, 1, (len(args) * 2) + len(fixedargs) + 1)
				fullarglist[0] = command
				for _, arg := range fixedargs {
					fullarglist = append(fullarglist, arg)
				}
				for argindex := 0; argindex < len(args); argindex += 1 {
					fullarglist = append(fullarglist, argnames[argindex])
					fullarglist = append(fullarglist, args[argindex])
				}

				cmd := &exec.Cmd{
					Path: command,
					Args: fullarglist,
					Stdout: os.Stdout,
					Stderr: os.Stderr,
				}

				err := cmd.Start()
				if err != nil {
					log.Printf("Failed to run %v: %v", fullarglist, err)
					continue
				}

				err = cmd.Wait()
				if err != nil {
					log.Printf("Failed to wait %v: %v", fullarglist, err)
					continue
				}
			}
			wg.Done()
		}()
	}

	// push data into the distribution channel - should be 
	// limited in pace based on how fast the workers drain the channel
	for {
		args, err := argreader.Read()
		if err != nil {
			break
		}
		distribution <- args
	}
	// kill the workers by sending a null for each one
	for i := 0; i < parallelism; i += 1 {
		distribution <- []string{}
	}
	wg.Wait()

	return nil
}

func main() {
	fmt.Printf("Assembling the merry tasks...\n")

	app := cli.App("littlejohn", "Run the same command with argument management")
	app.Spec = "[-j] -c PROG -- [ARGS...]"

	parallelism := app.IntOpt("j", 4, "Number of parallel copies to run")
	csv := app.StringOpt("c csv", "", "File with CSV arguments for program")
	command := app.StringArg("PROG", "", "Program to run")
	args := app.StringsArg("ARGS", nil, "Fixed arguments to all child programs")

	app.Action = func() {
		runner(*command, *parallelism, *csv, *args)
	}

	app.Run(os.Args)
}