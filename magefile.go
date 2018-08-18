// +build mage

package main

import (
	"fmt"
	"os/exec"
)

// Generate the mocks.
func Mock() error {
	cmd := exec.Command("go", "generate", "./...")
	return cmd.Run()
}

func Test() error {
	cmd := exec.Command("go", "test", "-failfast", "-race", "-cover", "-v", "./...")
	content, err := cmd.Output()
	if err != nil {
		return err
	}

	fmt.Println(string(content))
	return nil
}

// Check run all the linters to ensure the quality of the code.
func Check() error {

}
