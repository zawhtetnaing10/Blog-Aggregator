package main

import (
	"fmt"
	"os"

	"github.com/zawhtetnaing10/Blog-Aggregator/internal/config"
)

func main() {
	// Read
	currentConfig, err := config.Read()
	if err != nil {
		fmt.Println(err.Error())
	}

	currentState := config.State{
		Config: &currentConfig,
	}

	// Commands struct
	commands := config.Commands{
		CmdHandlers: make(map[string]func(*config.State, config.Command) error),
	}

	// Register Login Command
	commands.Register("login", config.LoginHandler)

	cmdArguments := os.Args
	if len(cmdArguments) < 2 {
		fmt.Println("not enough arguments were provided")
		os.Exit(1)
	}

	// Typed in user name
	commandName := cmdArguments[1]
	arguments := cmdArguments[2:]

	// Command struct
	command := config.Command{
		Name:      commandName,
		Arguments: arguments,
	}

	// Run the command
	if err := commands.Run(&currentState, command); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
