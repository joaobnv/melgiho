// This hook execute the tests and verify if all pass.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"time"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	testCmd := exec.CommandContext(ctx, "go", "test", "-json", "-cover", "-vet=all", path.Join(wd, "..."))

	outPipe, err := testCmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	errPipe, err := testCmd.StderrPipe()
	if err != nil {
		panic(err)
	}

	if err = testCmd.Start(); err != nil {
		panic(err)
	}

	go io.Copy(os.Stderr, errPipe)

	var fail bool
	var te TestEvent

	coverageRe := regexp.MustCompile(`^coverage: (\d{1,3}(?:\.\d)?)% of statements\n$`)

	dec := json.NewDecoder(outPipe)
	for {
		if err = dec.Decode(&te); err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		if te.Action == "fail" {
			if te.Test == "" {
				fmt.Printf("%s: tests failed\n", te.Package)
			} else {
				fmt.Printf("%s: test %s failed\n", te.Package, te.Test)
			}
			fail = true
		}

		if te.Action == "output" && coverageRe.MatchString(te.Output) {
			submatches := coverageRe.FindAllStringSubmatch(te.Output, -1)
			if submatches[0][1] != "100.0" {
				fmt.Printf("%s: test coverage is not 100.0%%\n", te.Package)
				fail = true
			}
		}
	}

	if err = testCmd.Wait(); err != nil {
		panic(err)
	}

	if fail {
		os.Exit(1)
	}
}

// TestEvent is a event generated by the test command.
type TestEvent struct {
	Time    time.Time
	Action  string
	Package string
	Test    string
	Elapsed float64
	Output  string
}
