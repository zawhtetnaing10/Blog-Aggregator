package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/zawhtetnaing10/Blog-Aggregator/internal/config"

	_ "github.com/lib/pq"
	"github.com/zawhtetnaing10/Blog-Aggregator/internal/database"
)

func main() {
	// Read from config file
	currentConfig, err := config.Read()
	if err != nil {
		fmt.Println(err.Error())
	}

	// Load Database
	db, err := sql.Open("postgres", currentConfig.DbUrl)
	if err != nil {
		fmt.Println(err.Error())
	}

	dbQueries := database.New(db)

	// Initialize State
	currentState := config.State{
		Config: &currentConfig,
		Db:     dbQueries,
	}

	// Commands struct
	commands := config.Commands{
		CmdHandlers: make(map[string]func(*config.State, config.Command) error),
	}

	// Register Login Command
	commands.Register("login", config.LoginHandler)
	commands.Register("register", config.RegisterHandler)
	commands.Register("reset", config.ResetHandler)
	commands.Register("users", config.UsersHandler)
	commands.Register("agg", config.AggHandler)

	cmdArguments := os.Args
	if len(cmdArguments) < 2 {
		fmt.Println("not enough arguments were provided")
		os.Exit(1)
	}

	// Separate command name and args for command structs
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
