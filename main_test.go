package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	timeout := 1 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	wg := new(sync.WaitGroup)

	commands := []Command{
		{
			Name:    "run 1",
			Command: "sh",
			Args:    []string{"-c", "echo 'hello'; echo 'world'; sleep 2"},
		},
		{
			Name:    "run 2",
			Command: "sh",
			Args:    []string{"-c", "echo 'hello'; echo 'world'; sleep 2"},
		},
	}
	lenCommands := len(commands)

	outputChan := make(chan Message, 2)

	for _, command := range commands {
		Execute(ctx, wg, outputChan, command)
	}

	messageCount := make(map[MessageType]int)
	mgs := make([]string, 0)

	wg.Wait()

	streamLogs(
		outputChan, lenCommands,
		func(message Message) {
			messageCount[message.Type] += 1
			if message.Content != "" {
				mgs = append(mgs, message.Content)
			}
		},
	)

	expectedMessageCount := map[MessageType]int{
		OutputStart:  2,
		OutputStdout: 4,
		OutputEnd:    2,
		SystemError:  2,
	}
	expectedMessages := []string{"hello", "world", "hello", "world", "error waiting for command: signal: killed", "error waiting for command: signal: killed"}
	assert.Equal(t, expectedMessageCount, messageCount)
	assert.Equal(t, expectedMessages, mgs)

	close(outputChan)
}
