package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mikelawson03/gator_bootdev/internal/database"
)

type commands struct {
	cmds map[string]func(*state, command) error
}

type command struct {
	name string
	args []string
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func (c *commands) run(s *state, cmd command) error {
	handler, exists := c.cmds[cmd.name]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}
	if err := handler(s, cmd); err != nil {
		return err
	}

	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.cmds[name] = f
}

func (f *RSSFeed) unescapeFeed() {
	f.Channel.Title = html.UnescapeString(f.Channel.Title)
	f.Channel.Description = html.UnescapeString((f.Channel.Description))

	for i := range f.Channel.Item {
		f.Channel.Item[i].Title = html.UnescapeString((f.Channel.Item[i].Title))
		f.Channel.Item[i].Description = html.UnescapeString(f.Channel.Item[i].Description)
	}
}

func fetchFeed(ctx context.Context, client *http.Client, feedURL string) (*RSSFeed, error) {
	feed := &RSSFeed{}

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return feed, err
	}

	req.Header.Set("User-Agent", "gator")
	res, err := client.Do(req)
	if err != nil {
		return feed, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return feed, err
	}

	if err := xml.Unmarshal(body, feed); err != nil {
		return feed, err
	}

	feed.unescapeFeed()

	return feed, nil

}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("Login command requires username. Syntax: go run . login <username>")
	}

	username := cmd.args[0]

	if _, err := s.db.GetUser(context.Background(), username); err != nil {
		return err
	}

	s.cfg.CurrentUserName = username
	if err := s.cfg.SetUser(); err != nil {
		return fmt.Errorf("Error setting user")
	}

	fmt.Println(s.cfg.CurrentUserName, "logged in successfully")

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("Register command requires username. Syntax: go run . register <username>")
	}

	p := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
	}

	u, err := s.db.CreateUser(context.Background(), p)
	if err != nil {
		return err
	}

	s.cfg.CurrentUserName = u.Name
	s.cfg.SetUser()
	fmt.Printf("User %v has been created\n", u.Name)
	fmt.Println(u)
	return nil
}

func handlerReset(s *state, cmd command) error {
	if err := s.db.Reset(context.Background()); err != nil {
		return err
	}
	fmt.Println("Table reset.")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	if len(users) == 0 {
		fmt.Println("No users found. Add users with go run . register <username>")
	}

	for _, user := range users {
		fmt.Print("* ", user.Name)
		if user.Name == s.cfg.CurrentUserName {
			fmt.Print(" (current)")
		}
		fmt.Print("\n")
	}

	return nil
}

func aggHandler(s *state, cmd command) error {
	f, err := fetchFeed(context.Background(), s.client, "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}

	fmt.Println(*f)
	return nil
}

func addFeedHandler(s *state, cmd command) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("addfeed command requires feed name and url. Syntax: go run . register <name> <url>")
	}

	user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return err
	}

	p := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
		Url:       cmd.args[1],
		UserID:    user.ID,
	}

	feed, err := s.db.CreateFeed(context.Background(), p)
	if err != nil {
		return err
	}

	fmt.Println(feed)

	return nil
}

func feedsHandler(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}

	for _, item := range feeds {
		user, err := s.db.GetUserById(context.Background(), item.UserID)
		if err != nil {
			return err
		}

		fmt.Printf("Feed name: %s\nURL: %s\nAdded by: %s\n\n", item.Name, item.Url, user.Name)
	}
	return nil
}
