package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/zawhtetnaing10/Blog-Aggregator/internal/database"
	"github.com/zawhtetnaing10/Blog-Aggregator/internal/network"
)

const configFileName = ".gatorconfig.json"

// State
type State struct {
	Config *Config
	Db     *database.Queries
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

// Handle Following
func FollowingHandler(s *State, cmd Command) error {
	// Current user name
	user_name := s.Config.CurrentUsername

	// Current user
	user, err := s.Db.GetUser(context.Background(), user_name)
	if err != nil {
		return fmt.Errorf("error getting user: %w", err)
	}

	// Get feed follows from db
	feedFollows, err := s.Db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("error getting feed follows: %w", err)
	}

	// Successfully print out the result
	fmt.Println("Following feeds:")
	for _, feedFollow := range feedFollows {
		fmt.Printf("  * %v\n", feedFollow.FeedName)
	}
	return nil
}

// Handle Follow
func FollowHandler(s *State, cmd Command) error {
	// early exit with error if command arguments are empty
	if len(cmd.Arguments) == 0 {
		return fmt.Errorf("you need to provide the feed url to follow")
	}

	// Feed url from command
	feed_url := cmd.Arguments[0]
	user_name := s.Config.CurrentUsername

	feed, err := s.Db.GetFeedByUrl(context.Background(), feed_url)
	if err != nil {
		return fmt.Errorf("error getting feed :%w", err)
	}

	user, err := s.Db.GetUser(context.Background(), user_name)
	if err != nil {
		return fmt.Errorf("error getting user: %w", err)
	}

	// Create feed follow param
	params := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}

	// Create feed follow
	feedFollowRow, err := s.Db.CreateFeedFollow(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error creating feed follow: %w", err)
	}

	// Print out the result and return nil
	fmt.Printf("Successfully followed feed \n user : %v \n feed : %v\n", feedFollowRow.UserName, feedFollowRow.FeedName)
	return nil
}

// Handle Feeds
func FeedsHandler(s *State, cmd Command) error {
	// Get Feeds from DB
	feeds, err := s.Db.GetFeedsWithUsername(context.Background())
	if err != nil {
		return fmt.Errorf("error fetching feeds from db: %w", err)
	}

	// Print out feed information
	for index, feed := range feeds {
		fmt.Printf("Feed : %v\n", index+1)
		fmt.Printf("  * %v\n", feed.Name)
		fmt.Printf("  * %v\n", feed.Url)
		fmt.Printf("  * %v\n", feed.Username)
	}

	return nil
}

// Handle Add Feed
func AddFeedHandler(s *State, cmd Command) error {
	// early exit with error if command arguments are empty
	if len(cmd.Arguments) <= 1 {
		return fmt.Errorf("your need to provide both name and url to post a feed")
	}

	// Get name and url
	name := cmd.Arguments[0]
	url := cmd.Arguments[1]

	// Get current logged in user name
	loggedInUserName := s.Config.CurrentUsername
	loggedInUser, err := s.Db.GetUser(context.Background(), loggedInUserName)
	if err != nil {
		return err
	}

	// Create Feed Params
	feedParams := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    uuid.NullUUID{UUID: loggedInUser.ID, Valid: true},
	}

	// Insert feed into database
	insertedFeed, err := s.Db.CreateFeed(context.Background(), feedParams)
	if err != nil {
		return fmt.Errorf("error inserting feed into db: %w", err)
	}

	// Create feed follow param
	feedFollowParams := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    loggedInUser.ID,
		FeedID:    insertedFeed.ID,
	}
	_, feedFollowErr := s.Db.CreateFeedFollow(context.Background(), feedFollowParams)
	if feedFollowErr != nil {
		return fmt.Errorf("error creating feed follow: %w", feedFollowErr)
	}

	// Print out the inserted feed
	fmt.Printf("%v\n", insertedFeed)

	return nil
}

// Agg Handler
func AggHandler(s *State, cmd Command) error {
	// Make the api request
	result, err := network.FetchFeed(context.Background(), network.RSS_FEED_URL)
	if err != nil {
		return err
	}

	// Print out the whole feed struct
	fmt.Printf("%v\n", result)

	return nil
}

// Users Handler
func UsersHandler(s *State, cmd Command) error {
	users, err := s.Db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error fetching users: %w", err)
	}

	loggedInUserName := s.Config.CurrentUsername

	for _, user := range users {
		if loggedInUserName == user.Name {
			fmt.Printf("* %v (current)\n", user.Name)
		} else {
			fmt.Printf("* %v\n", user.Name)
		}
	}

	return nil
}

// Handle Reset
func ResetHandler(s *State, cmd Command) error {
	if err := s.Db.ResetUsers(context.Background()); err != nil {
		return fmt.Errorf("error resetting users %w", err)
	}

	fmt.Println("All data has been reset")

	return nil
}

// Handle Register
func RegisterHandler(s *State, cmd Command) error {
	// early exit with error if command arguments are empty
	if len(cmd.Arguments) == 0 {
		return fmt.Errorf("you need to provide a username to login")
	}

	// Get the name from command
	name := cmd.Arguments[0]

	// Return error if already exists
	_, err := s.Db.GetUser(context.Background(), name)
	if err == nil {
		// User exists, exit
		return fmt.Errorf("user already exists")
	}

	// Create the params to save to db
	createUserParams := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
	}
	// Save to db
	createdUser, err := s.Db.CreateUser(context.Background(), createUserParams)
	if err != nil {
		return fmt.Errorf("error creating user %w", err)
	}

	// Update config
	s.Config.CurrentUsername = createdUser.Name

	// Write the config to json file
	if err := s.SaveConfig(); err != nil {
		return fmt.Errorf("error saving config %w", err)
	}

	// Success message
	fmt.Println("user has been created")

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

	// Get user from db
	user, err := s.Db.GetUser(context.Background(), username)
	if err != nil {
		return fmt.Errorf("user not found %w", err)
	}

	// Set username to State
	s.Config.CurrentUsername = user.Name

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
