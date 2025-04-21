package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

// State
type State struct {
	Config *Config
}

// Write the state back to the config file
func (s *State) SaveConfig() error {
	return write(*s.Config)
}

// Command
type Command struct {
	Name      string
	Arguments []string
}

// Config
type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUsername string `json:"current_user_name"`
}

// Commands
type Commands struct {
	CmdHandlers map[string]func(*State, Command) error
}

// Registers a command to commands struct
func (c *Commands) Register(name string, f func(*State, Command) error) {
	c.CmdHandlers[name] = f
}

// Run the command function according to the given command
func (c *Commands) Run(s *State, cmd Command) error {
	// Get the corresponding function in Commands
	commandToRun, ok := c.CmdHandlers[cmd.Name]
	if !ok {
		return fmt.Errorf("command does not exist")
	}

	// Run the command and check for errors in the same line
	if err := commandToRun(s, cmd); err != nil {
		return err
	}

	return nil
}

// Handle login
func LoginHandler(s *State, cmd Command) error {
	// early exit with error if command arguments are empty
	if len(cmd.Arguments) == 0 {
		return fmt.Errorf("you need to provide a username to login")
	}

	// Get user name
	username := cmd.Arguments[0]

	// Set username to State
	s.Config.CurrentUsername = username

	// Write the config to json file
	if err := s.SaveConfig(); err != nil {
		return fmt.Errorf("error saving config %w", err)
	}

	// Prints message
	fmt.Println("user has been set")

	return nil
}

// Get the configuration file path
func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory %w", err)
	}
	return filepath.Join(homeDir, configFileName), nil
}

// Read gatorconfig.json and return the populated Config struct
func Read() (Config, error) {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return Config{}, fmt.Errorf("error reading json file %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("error parsing json %w", err)
	}

	return config, nil
}

// Setting user to gatorconfig.json
func (c *Config) SetUser(username string) error {
	c.CurrentUsername = username
	err := write(*c)
	return err
}

// Write the config file
func write(config Config) error {
	bytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling data %w", err)
	}

	configFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	writeErr := os.WriteFile(configFilePath, bytes, 0644)
	if writeErr != nil {
		return fmt.Errorf("error writing file %w", err)
	}

	return nil
}
