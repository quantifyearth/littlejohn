package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	cli "github.com/jawher/mow.cli"
)

func openCSV(filename string) (*csv.Reader, []string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	reader := csv.NewReader(file)

	raw_argnames, err := reader.Read()
	if err != nil {
		return nil, nil, err
	}

	// for the arguments, strip any extraneous spaces
	argnames := make([]string, len(raw_argnames))
	for idx, value := range raw_argnames {
		argnames[idx] = strings.TrimSpace(value)
	}

	return reader, argnames, nil
}

func run_command(command string, fullarglist []string, output_channel chan string) {

	cmd := &exec.Cmd{
		Path:   command,
		Args:   fullarglist,
		Stderr: os.Stderr,
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to get output pipe for %v: %v", fullarglist, err)
		return
	}
	defer stdout.Close()

	err = cmd.Start()
	if err != nil {
		log.Printf("Failed to run %v: %v", fullarglist, err)
		return
	}

	buf := bufio.NewReader(stdout)
	count := 0
	for {
		line, _, err := buf.ReadLine()
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading process %v: %v", fullarglist, err)
			}
			break
		}
		result := fullarglist[0]
		for _, arg := range fullarglist[1:] {
			result = fmt.Sprintf("%v, %v", result, arg)
		}
		result = fmt.Sprintf("%v, %d, %v", result, count, string(line))
		output_channel <- result
		count += 1
	}

	err = cmd.Wait()
	if err != nil {
		log.Printf("Failed to wait %v: %v", fullarglist, err)
		return
	}
}

func cmd_main(command string, parallelism int, dryrun bool, csvfile string, fixedargs []string, outputfile string) error {

	argreader, argnames, err := openCSV(csvfile)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}

	var output io.Writer = os.Stdout
	if outputfile != "" {
		f, err := os.Create(outputfile)
		if err != nil {
			return fmt.Errorf("Failed to open output file: %w", err)
		}
		defer f.Close()
		output = f
	}

	output_channel := make(chan string, 1)
	go func() {
		for {
			res := <-output_channel
			output.Write([]byte(res + "\n"))
		}
	}()

	distribution := make(chan []string, parallelism)

	wg := new(sync.WaitGroup)

	for i := 0; i < parallelism; i += 1 {
		wg.Add(1)
		go func() {
			for {
				args := <-distribution
				if len(args) == 0 {
					break
				}

				if len(args) != len(argnames) {
					log.Printf("Skipping run as %v and %v don't match", args, argnames)
					continue
				}

				fullarglist := make([]string, 1, (len(args)*2)+len(fixedargs)+1)
				fullarglist[0] = command
				fullarglist = append(fullarglist, fixedargs...)
				for argindex := 0; argindex < len(args); argindex += 1 {
					fullarglist = append(fullarglist, argnames[argindex])
					fullarglist = append(fullarglist, args[argindex])
				}

				if dryrun {
					fmt.Printf("%s\n", strings.Join(fullarglist, " "))
					continue
				}
				fmt.Printf("%s\n", strings.Join(fullarglist, " "))

				run_command(command, fullarglist, output_channel)
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
	app.Spec = "[-jno] -c PROG -- [ARGS...]"

	parallelism := app.IntOpt("j", 4, "Number of parallel copies to run")
	dryrun := app.BoolOpt("n dry-run", false, "Just print out commands rather than run")
	csv := app.StringOpt("c csv", "", "File with CSV arguments for program")
	command := app.StringArg("PROG", "", "Program to run")
	output := app.StringOpt("o outputfile", "", "File to write output to")
	args := app.StringsArg("ARGS", nil, "Fixed arguments to all child programs")

	app.Action = func() {
		cmd_main(*command, *parallelism, *dryrun, *csv, *args, *output)
	}

	app.Run(os.Args)
}
