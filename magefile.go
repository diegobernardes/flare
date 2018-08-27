// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	spin "github.com/tj/go-spin"
)

// Generate the mocks.
func Mock() error {
	cmd := exec.Command("go", "generate", "./...")
	return cmd.Run()
}

func Test() error {
	verbose := (os.Getenv("verbose") == "true")

	go func() {
		s := spin.New()
		for {
			fmt.Printf("\r  \033[36mtesting\033[m %s ", s.Next())
			time.Sleep(100 * time.Millisecond)
		}
	}()

	cmd := exec.Command("go", "test", "-failfast", "-race", "-cover", "-v", "./...")
	content, err := cmd.Output()
	if err != nil {
		return err
	}

	if verbose {
		fmt.Println(string(content))
	}
	return nil
}

// Check run all the linters to ensure the quality of the code.
func Check() error {
	return nil
}
