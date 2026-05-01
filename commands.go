package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("addfeed command requires feed name and url. Syntax: go run . addfeed <name> <url>")
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

	fmt.Println("New feed", feed.Name, "added.")

	new_follow := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}

	res, err := s.db.CreateFeedFollow(context.Background(), new_follow)
	if err != nil {
		return err
	}

	fmt.Println("Current user", res.UserName, "now following", res.FeedName)

	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("agg command requires duration. Syntax: go run . agg <duration>")
	}

	t, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Collecting feeds every %s\n", t.String())

	ticker := time.NewTicker(t)

	for ; ; <-ticker.C {

		if err := scrapeFeeds(s); err != nil {
			fmt.Println(err)
		}

	}
}

func handlerBrowse(s *state, cmd command) error {
	user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return err
	}

	p := database.GetPostsByUserIdParams{
		UserID: user.ID,
	}

	if len(cmd.args) < 1 {
		p.Limit = 2
	}
	if len(cmd.args) > 0 {
		l, err := strconv.Atoi(cmd.args[0])
		if err != nil {
			return err
		}
		var limit interface{} = l
		_, ok := limit.(int)
		if !ok {
			return fmt.Errorf("browse's optional limit must be an integer. Syntax: go run . browse <limit>")
		}
		p.Limit = int32(l)
	}

	posts, err := s.db.GetPostsByUserId(context.Background(), p)
	if err != nil {
		return err
	}

	// fmt.Print(posts)

	for _, item := range posts {
		fmt.Println(item.Title)
		fmt.Println(item.Url)
		fmt.Println(item.Description)
		fmt.Println(item.PublishedAt)
		fmt.Println("--------------------")
		fmt.Println()
	}

	return nil
}

func handlerFeeds(s *state, cmd command) error {
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

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("follow command requires feed url. Syntax: go run . follow <url>")
	}

	feedid, err := s.db.GetFeedByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}

	p := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feedid.ID,
	}

	res, err := s.db.CreateFeedFollow(context.Background(), p)

	fmt.Println(res)
	fmt.Printf("User %s now following feed %s\n", res.UserName, res.FeedName)

	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {

	res, err := s.db.GetFollowsByUserId(context.Background(), user.ID)
	if err != nil {
		return err
	}

	fmt.Println("Current follows for", user.Name)

	for _, row := range res {
		fmt.Println(row.FeedName)
	}

	return nil
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

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("unfollow command requires feed url. Syntax: go run . unfollow <url>")
	}
	res, err := s.db.GetFeedByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}

	p := database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: res.ID,
	}

	if err = s.db.DeleteFeedFollow(context.Background(), p); err != nil {
		return err
	}

	fmt.Printf("Feed %s unfollowed by %s\n", res.Name, user.Name)

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

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return err
		}

		return handler(s, cmd, user)
	}
}

func scrapeFeeds(s *state) error {
	res, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	feed, err := fetchFeed(context.Background(), s.client, res.Url)
	if err != nil {
		return err
	}

	fmt.Println(feed.Channel.Title)

	p := database.MarkFeedFetchedParams{
		LastFetchedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true},
		UpdatedAt: time.Now(),
		ID:        res.ID,
	}

	s.db.MarkFeedFetched(context.Background(), p)

	fmt.Printf("Saving posts from %s\n", feed.Channel.Title)

	for _, item := range feed.Channel.Item {
		if err := savePost(s, item, res.ID); err != nil {
			fmt.Println(err)
		}
	}

	return nil
}

func savePost(s *state, item RSSItem, feedId uuid.UUID) error {
	pubDate, err := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", item.PubDate)
	if err != nil {
		return err
	}

	p := database.CreatePostParams{
		ID:          uuid.New(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Title:       item.Title,
		Url:         item.Link,
		Description: item.Description,
		PublishedAt: pubDate,
		FeedID:      feedId,
	}

	post, err := s.db.CreatePost(context.Background(), p)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return nil
			}
		}
		return err
	}

	fmt.Printf("Post '%s' saved to database\n", post.Title)

	return nil

}
