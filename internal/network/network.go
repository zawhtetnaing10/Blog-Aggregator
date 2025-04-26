package network

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"
)

// RSS Feed Url
const RSS_FEED_URL = "https://www.wagslane.dev/index.xml"

// Rss Feed object
type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

// Rss Channel
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// Fetch RSS Feeds
func FetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {

	// Create client
	client := &http.Client{Timeout: 5 * time.Second}

	// Create new request
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("error creating request: %w", err)
	}

	// Set header
	req.Header.Set("User-Agent", "gator")

	// Make request
	res, err := client.Do(req)
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("error making the request %w", err)
	}
	defer res.Body.Close()

	// Read the response
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("error reading the response %w", err)
	}

	// Parse xml
	var result RSSFeed
	if err := xml.Unmarshal(body, &result); err != nil {
		return &RSSFeed{}, fmt.Errorf("error parsing the response %w", err)
	}

	// Unescape html and mutate the resulting feed
	unEscapeHtml(&result)

	return &result, nil
}

// Unescape html for title and description
func unEscapeHtml(feed *RSSFeed) {
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	for i := 0; i < len(feed.Channel.Item); i++ {
		feed.Channel.Item[i].Title = html.UnescapeString(feed.Channel.Item[i].Title)
		feed.Channel.Item[i].Description = html.UnescapeString(feed.Channel.Item[i].Description)
	}
}
