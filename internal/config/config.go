package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/zawhtetnaing10/Blog-Aggregator/internal/constants"
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

// Midel ware logged in
func MiddlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(s *State, cmd Command) error {
	return func(s *State, cmd Command) error {
		user, err := s.Db.GetUser(context.Background(), s.Config.CurrentUsername)

		if err != nil {
			return fmt.Errorf("error fetching user: %w", err)
		}

		return handler(s, cmd, user)
	}
}

// Browse Handler
func BrowseHandler(s *State, cmd Command, user database.User) error {
	var limit int
	if len(cmd.Arguments) == 0 {
		limit = 2
	} else {
		convertedInt, err := strconv.ParseInt(cmd.Arguments[0], 10, 32)
		if err != nil {
			return fmt.Errorf("error converting the input: %w", err)
		}
		limit = int(convertedInt)
	}

	params := database.GetPostsForUserParams{
		UserID: uuid.NullUUID{
			UUID:  user.ID,
			Valid: true,
		},
		Limit: int32(limit),
	}

	posts, err := s.Db.GetPostsForUser(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error fetching posts from db: %w", err)
	}

	fmt.Println("Followed posts : ")
	for _, post := range posts {
		fmt.Printf(" * %v\n", post.Title)
	}

	return nil
}

// Unfollow Handler
func UnfollowHandler(s *State, cmd Command, user database.User) error {
	// early exit with error if command arguments are empty
	if len(cmd.Arguments) == 0 {
		return fmt.Errorf("you need to provide the feed url to unfollow")
	}
	feedUrl := cmd.Arguments[0]
	feed, err := s.Db.GetFeedByUrl(context.Background(), feedUrl)
	if err != nil {
		return fmt.Errorf("error fetching feed: %w", err)
	}

	params := database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}

	// Delete the feed follow entry
	result, err := s.Db.DeleteFeedFollow(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error deleting feed: %w", err)
	}

	// Get the rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error fetching the deleted feed: %w", err)
	}
	// If rows affected is 0, the feed is unfollowed. Create an error and exit
	if rowsAffected == 0 {
		return fmt.Errorf("you need to have followed the feed in the first place to unfollow")
	}

	// print successful message
	fmt.Println("successfully unfollowed the feed")
	return nil
}

// Handle Following
func FollowingHandler(s *State, cmd Command, user database.User) error {
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
func FollowHandler(s *State, cmd Command, user database.User) error {
	// early exit with error if command arguments are empty
	if len(cmd.Arguments) == 0 {
		return fmt.Errorf("you need to provide the feed url to follow")
	}

	// Feed url from command
	feed_url := cmd.Arguments[0]

	feed, err := s.Db.GetFeedByUrl(context.Background(), feed_url)
	if err != nil {
		return fmt.Errorf("error getting feed :%w", err)
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
func AddFeedHandler(s *State, cmd Command, user database.User) error {
	// early exit with error if command arguments are empty
	if len(cmd.Arguments) <= 1 {
		return fmt.Errorf("your need to provide both name and url to post a feed")
	}

	// Get name and url
	name := cmd.Arguments[0]
	url := cmd.Arguments[1]

	// Create Feed Params
	feedParams := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    uuid.NullUUID{UUID: user.ID, Valid: true},
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
		UserID:    user.ID,
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

	// Early exit if time between requests is not provided
	if len(cmd.Arguments) == 0 {
		return fmt.Errorf("time between requests must be provided")
	}

	// Get time string
	time_string := cmd.Arguments[0]

	// Parse the input time into duration
	time_between_reqs, err := time.ParseDuration(time_string)
	if err != nil {
		return fmt.Errorf("error parsing the time given: %w", err)
	}

	// Collecting feeds message
	fmt.Printf("Collecting feed every %v\n", time_string)

	// Print out the feeds to console
	ticker := time.NewTicker(time_between_reqs)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

// Scrape Feeds. Fetch feeds from network and print them out to console.
func scrapeFeeds(s *State) error {
	// Get next feed to fetch
	nextFeed, err := s.Db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get next feed to fetch: %w", err)
	}

	// Mark it as fetched
	params := database.MarkFeedFetchedParams{
		LastFetchedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
		UpdatedAt: time.Now(),
		ID:        nextFeed.ID,
	}
	if err := s.Db.MarkFeedFetched(context.Background(), params); err != nil {
		return fmt.Errorf("error marking feed as fetched: %w", err)
	}

	// Fetch feeds from network using the url
	fetchedFeeds, err := fetchFeedsFromNetwork(nextFeed.Url)
	if err != nil {
		return fmt.Errorf("error fetching feeds from network: %w", err)
	}

	// Print out the results
	fmt.Printf("Fetched feeds from %v: \n", fetchedFeeds.Channel.Title)
	for _, post := range fetchedFeeds.Channel.Item {
		// Parse the date and save the post only when parsing is successful
		parsedDate, err := parseDate(post.PubDate)
		if err == nil {
			// Parsing successful, save the post
			params := database.CreatePostParams{
				ID:          uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Url:         post.Link,
				Title:       post.Title,
				Description: post.Description,
				PublishedAt: parsedDate,
				FeedID:      nextFeed.ID,
			}

			// Add the post to db
			_, createPostErr := s.Db.CreatePost(context.Background(), params)
			if createPostErr != nil {
				// If error is not nil, check if its unique constraint violaton
				// Only when it's not, print out the error
				errCode := createPostErr.(*pq.Error).Code
				if errCode != constants.ERR_CODE_UNIQUE_CONSTRAINT_VIOLATION {
					fmt.Printf("Error saving post: %v", createPostErr.Error())
				}
			}
		} else {
			return fmt.Errorf("error parsing date: %w", err)
		}
	}

	fmt.Println("Successfully fetched the posts and saved.")

	return nil
}

// Parse date from server
func parseDate(date string) (time.Time, error) {
	// Try multiple formats in sequence
	formats := []string{
		time.RFC1123Z,                    // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC1123,                     // "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC822Z,                     // "02 Jan 06 15:04 -0700"
		time.RFC822,                      // "02 Jan 06 15:04 MST"
		"2006-01-02T15:04:05Z07:00",      // ISO8601 / RFC3339
		"2006-01-02T15:04:05-07:00",      // ISO8601 variant
		"2006-01-02 15:04:05",            // Simple datetime
		"Mon, 2 Jan 2006 15:04:05 -0700", // RFC1123Z with single-digit day
	}

	var publishedTime time.Time
	var err error
	for _, format := range formats {
		publishedTime, err = time.Parse(format, date)
		if err == nil {
			break
		}
	}

	// error parsing time.
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing time: %w", err)
	}

	return publishedTime, nil
}

// Utility function to fetch feeds from Network
func fetchFeedsFromNetwork(url string) (*network.RSSFeed, error) {
	// Make the api request
	result, err := network.FetchFeed(context.Background(), url)
	if err != nil {
		return &network.RSSFeed{}, err
	}

	return result, nil
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
