package main

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCLI(t *testing.T) {
	cli := NewCLI()
	assert.NotNil(t, cli)
	assert.NotNil(t, cli.workerPool)
	assert.NotNil(t, cli.notificationEngine)
}

func TestCLI_Run_ListModels(t *testing.T) {
	cli := NewCLI()

	// Set flags
	oldArgs := os.Args
	os.Args = []string{"cli", "--list-models"}
	defer func() { os.Args = oldArgs }()

	// Reset flag state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Execute
	err := cli.Run()

	// Assertions
	assert.NoError(t, err)
}
