package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"gopkg.in/yaml.v3"
)

// Config represents the configuration structure loaded from a YAML file.
type Config struct {
	Version string    `yaml:"version"`
	Apps    []Command `yaml:"apps"`
}

// Command represents a system command to be executed.
type Command struct {
	// Name is a descriptive name for the command.
	Name string `yaml:"name"`
	// Command is the actual system command to be executed.
	Command string `yaml:"command"`
	// Args are the arguments to be passed to the command.
	Args []string `yaml:"args"`
}

// MessageType represents the type of message.
type MessageType int

// Name returns the name of the MessageType.
func (m MessageType) Name() string {
	switch m {
	case OutputStart:
		return "OutputStart"
	case OutputStdout:
		return "OutputStdout"
	case OutputStderr:
		return "OutputStderr"
	case OutputEnd:
		return "OutputEnd"
	case SystemError:
		return "SystemError"
	}
	return "Unknown"
}

// Message types
const (
	OutputStart  MessageType = iota // OutputStart indicates the start of command output.
	OutputStdout                    // OutputStdout indicates stdout output from the command.
	OutputStderr                    // OutputStderr indicates stderr output from the command.
	OutputEnd                       // OutputEnd indicates the end of command output.
	SystemError                     // SystemError indicates an error related to the system or command execution.
)

// Message represents a message containing the content, type, and associated command.
type Message struct {
	// Content is the message content.
	Content string
	// Type is the type of the message.
	Type MessageType
	// Command is the associated command.
	Command *Command
}

// CommandName returns the name of the associated command, or "system" if no command is present.
func (m Message) CommandName() string {
	if m.Command != nil {
		return m.Command.Name
	}
	return "system"
}

// Execute executes the given command in a separate goroutine.
// It captures the command output and sends it to the outputChan.
// It also handles errors and sends error messages to the outputChan.
func Execute(ctx context.Context, wg *sync.WaitGroup, outputChan chan<- Message, command Command) {
	go func(ctx context.Context, wg *sync.WaitGroup, outputChan chan<- Message, command Command) {
		// Defer wg.Done to ensure it is called even if the goroutine panics
		wg.Add(1)
		defer wg.Done()

		outputChan <- Message{
			Type:    OutputStart,
			Command: &command,
		}
		defer func() {
			outputChan <- Message{
				Type:    OutputEnd,
				Command: &command,
			}
		}()

		// Execute system command with context
		cmd := exec.CommandContext(ctx, command.Command, command.Args...)

		// Create pipes to capture stdout and stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			outputChan <- Message{
				Content: fmt.Errorf("error creating StdoutPipe: %w", err).Error(),
				Type:    SystemError,
			}
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			outputChan <- Message{
				Content: fmt.Errorf("error creating StderrPipe: %w", err).Error(),
				Type:    SystemError,
			}
			return
		}

		// Capture stdout and stderr output
		captureOutput(ctx, stdout, outputChan, command, OutputStdout)
		captureOutput(ctx, stderr, outputChan, command, OutputStderr)

		// Start the command
		err = cmd.Start()
		if err != nil {
			outputChan <- Message{
				Content: fmt.Errorf("error starting command: %w", err).Error(),
				Type:    SystemError,
			}
			return
		}

		// Wait for the command to finish
		err = cmd.Wait()
		if err != nil {
			outputChan <- Message{
				Content: fmt.Errorf("error waiting for command: %w", err).Error(),
				Type:    SystemError,
			}
		}
	}(ctx, wg, outputChan, command)
}

// captureOutput captures the output from the given io.ReadCloser and sends it to the outputChan.
// It runs in a separate goroutine and stops when the context is canceled or when the io.ReadCloser is closed.
func captureOutput(ctx context.Context, std io.ReadCloser, outputChan chan<- Message, command Command, messageType MessageType) {
	stdScanner := bufio.NewScanner(std)
	go func() {
		for stdScanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				// Send the line to the output channel
				outputChan <- Message{
					Content: stdScanner.Text(),
					Type:    messageType,
					Command: &command,
				}
			}
		}
	}()
}

// streamLogs streams log messages from the output channel and invokes the callback function for each message.
// It waits for all commands to complete before returning.
func streamLogs(outputChan <-chan Message, amountOfCommands int, callback func(message Message)) {
	for message := range outputChan {
		callback(message)

		// Check if the message type is OutputEnd
		if message.Type == OutputEnd {
			// Decrement the amountOfCommands counter
			amountOfCommands--

			// Check if all commands have completed and exit the function
			if amountOfCommands == 0 {
				return
			}
		}
	}
}

// loadConfig loads the configuration from a YAML file.
// It expects the path to the config file as a command-line argument.
// If the file is valid and the version is supported, it returns a Config object.
// Otherwise, it returns an error.
func loadConfig() (*Config, error) {
	// Check if the correct number of command-line arguments is provided
	if len(os.Args) != 2 {
		return nil, fmt.Errorf("usage: %s <config_file.yml>", os.Args[0])
	}

	// Get the config file path from the command-line argument
	configFilePath := os.Args[1]

	// Read the content of the config file
	configFileContent, err := os.ReadFile(configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file does not exist: %w", err)
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Unmarshal the YAML content into a Config object
	var config Config
	err = yaml.Unmarshal(configFileContent, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML content: %w", err)
	}

	// Check if the config version is supported
	if config.Version != "1" {
		return nil, errors.New("unsupported config version")
	}

	return &config, nil
}

func main() {
	// Load the configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Create a context and a cancel function for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Set up signal handling for interrupts and termination signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine to handle signals and cancel the context on signal reception
	go func() {
		<-sigs
		cancel()
	}()

	// Create a wait group to wait for all commands to complete
	wg := new(sync.WaitGroup)

	// Create a channel to receive output messages from commands
	outputChan := make(chan Message, 2)
	defer close(outputChan)

	// Execute each command concurrently
	commands := config.Apps
	amountOfCommands := len(commands)
	for _, command := range commands {
		Execute(ctx, wg, outputChan, command)
	}

	// Stream logs from the output channel and process them with a handler function
	streamLogs(
		outputChan, amountOfCommands,
		func(message Message) {
			log.Printf(
				"[%s::%s]: %s",
				message.CommandName(),
				message.Type.Name(),
				message.Content,
			)
		},
	)

	// Wait for all commands to complete
	wg.Wait()
}
