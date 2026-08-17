package command

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Zallu35/blog-aggregator/internal/app_state"
	"github.com/Zallu35/blog-aggregator/internal/database"
	"github.com/Zallu35/blog-aggregator/internal/rss"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Command struct {
	Name string
	Args []string
}

type Commands struct {
	CommandMap map[string]func(*app_state.AppState, Command) error
}

func (c *Commands) Run(s *app_state.AppState, cmd Command) error {
	handler, ok := c.CommandMap[cmd.Name]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmd.Name)
	}
	return handler(s, cmd)
}

func (c *Commands) Register(name string, f func(*app_state.AppState, Command) error) {
	if c.CommandMap == nil {
		c.CommandMap = make(map[string]func(*app_state.AppState, Command) error)
	}
	c.CommandMap[name] = f
}

func MiddlewareLoggedIn(handler func(*app_state.AppState, Command, database.User) error) func(*app_state.AppState, Command) error {
	return func(s *app_state.AppState, cmd Command) error {
		user, err := s.DBConn.GetUserByName(context.Background(), s.Config.CurrentUserName)
		if err != nil {
			fmt.Printf("Error fetching current user '%s': %v\n", s.Config.CurrentUserName, err)
			os.Exit(1)
		}

		return handler(s, cmd, user)
	}
}

func HandlerLogin(s *app_state.AppState, cmd Command) error {
	if len(cmd.Args) == 0 {
		return errors.New("the login handler expects a single argument: the username")
	}

	username := cmd.Args[0]

	_, err := s.DBConn.GetUserByName(context.Background(), username)
	if err != nil {
		fmt.Printf("Error fetching user '%s': %v\n", username, err)
		os.Exit(1)
	}

	err = s.Config.SetUser(username)
	if err != nil {
		return fmt.Errorf("error setting user in config: %w", err)
	}

	fmt.Printf("User has been set to: %s\n", username)
	return nil
}

func HandlerRegister(s *app_state.AppState, cmd Command) error {
	if len(cmd.Args) == 0 {
		return errors.New("the register handler expects a single argument: the username")
	}

	username := cmd.Args[0]

	now := time.Now()
	userID := uuid.New()

	user, err := s.DBConn.CreateUser(context.Background(), database.CreateUserParams{
		ID:        userID,
		CreatedAt: now,
		UpdatedAt: now,
		Name:      username,
	})
	if err != nil {
		// A more robust error check for duplicate user would be needed in a real application
		// For now, we'll assume a generic error means the user might already exist or similar DB issue.
		fmt.Printf("Error creating user: %v\n", err)
		os.Exit(1) // Exit with code 1 if user creation fails
	}

	err = s.Config.SetUser(username)
	if err != nil {
		return fmt.Errorf("error setting current user in config: %w", err)
	}

	fmt.Printf("User '%s' created successfully!\n", username)
	fmt.Printf("User data: %+v\n", user)

	return nil
}

func HandlerUsers(s *app_state.AppState, cmd Command) error {
	users, err := s.DBConn.GetUsers(context.Background())
	if err != nil {
		fmt.Printf("Error fetching users: %v\n", err)
		os.Exit(1)
	}

	for _, user := range users {
		if user.Name == s.Config.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

func HandlerAgg(s *app_state.AppState, cmd Command) error {
	if len(cmd.Args) == 0 {
		return errors.New("the agg handler expects a single argument: time_between_reqs")
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("error parsing time_between_reqs: %w", err)
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		if err := scrapeFeeds(s); err != nil {
			fmt.Printf("Error scraping feeds: %v\n", err)
		}
	}
}

func scrapeFeeds(s *app_state.AppState) error {
	feed, err := s.DBConn.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("error fetching next feed: %w", err)
	}

	if err := s.DBConn.MarkFeedFetched(context.Background(), feed.ID); err != nil {
		return fmt.Errorf("error marking feed fetched: %w", err)
	}

	rssFeed, err := rss.FetchFeed(context.Background(), feed.Url)
	if err != nil {
		return fmt.Errorf("error fetching feed '%s': %w", feed.Url, err)
	}

	for _, item := range rssFeed.Channel.Item {
		_, err := s.DBConn.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       item.Title,
			Url:         item.Link,
			Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
			PublishedAt: parsePubDate(item.PubDate),
			FeedID:      feed.ID,
		})
		if err != nil {
			if isDuplicatePostError(err) {
				continue
			}
			fmt.Printf("Error saving post '%s': %v\n", item.Title, err)
			continue
		}
	}

	return nil
}

var pubDateFormats = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC3339,
	"2006-01-02T15:04:05Z",
	"2006-01-02 15:04:05",
}

func parsePubDate(pubDate string) sql.NullTime {
	for _, format := range pubDateFormats {
		if t, err := time.Parse(format, pubDate); err == nil {
			return sql.NullTime{Time: t, Valid: true}
		}
	}
	return sql.NullTime{}
}

func isDuplicatePostError(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code == "23505"
}

func HandlerAddFeed(s *app_state.AppState, cmd Command, user database.User) error {
	if len(cmd.Args) < 2 {
		return errors.New("the addfeed handler expects two arguments: the feed name and url")
	}

	name := cmd.Args[0]
	url := cmd.Args[1]

	now := time.Now()

	feed, err := s.DBConn.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})
	if err != nil {
		fmt.Printf("Error creating feed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Feed created successfully!\n")
	fmt.Printf("Feed data: %+v\n", feed)

	feedFollow, err := createFeedFollow(s, user.ID, feed.ID)
	if err != nil {
		fmt.Printf("Error creating feed follow: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("'%s' is now following '%s'\n", feedFollow.UserName, feedFollow.FeedName)

	return nil
}

func HandlerFollow(s *app_state.AppState, cmd Command, user database.User) error {
	if len(cmd.Args) == 0 {
		return errors.New("the follow handler expects a single argument: the feed url")
	}

	url := cmd.Args[0]

	feed, err := s.DBConn.GetFeedByUrl(context.Background(), url)
	if err != nil {
		fmt.Printf("Error fetching feed '%s': %v\n", url, err)
		os.Exit(1)
	}

	feedFollow, err := createFeedFollow(s, user.ID, feed.ID)
	if err != nil {
		fmt.Printf("Error creating feed follow: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("'%s' is now following '%s'\n", feedFollow.UserName, feedFollow.FeedName)

	return nil
}

func HandlerFollowing(s *app_state.AppState, cmd Command, user database.User) error {
	feedFollows, err := s.DBConn.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		fmt.Printf("Error fetching feed follows: %v\n", err)
		os.Exit(1)
	}

	for _, feedFollow := range feedFollows {
		fmt.Printf("* %s\n", feedFollow.FeedName)
	}

	return nil
}

func HandlerUnfollow(s *app_state.AppState, cmd Command, user database.User) error {
	if len(cmd.Args) == 0 {
		return errors.New("the unfollow handler expects a single argument: the feed url")
	}

	url := cmd.Args[0]

	feed, err := s.DBConn.GetFeedByUrl(context.Background(), url)
	if err != nil {
		fmt.Printf("Error fetching feed '%s': %v\n", url, err)
		os.Exit(1)
	}

	err = s.DBConn.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		fmt.Printf("Error deleting feed follow: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("'%s' has unfollowed '%s'\n", user.Name, feed.Name)

	return nil
}

func createFeedFollow(s *app_state.AppState, userID, feedID uuid.UUID) (database.CreateFeedFollowRow, error) {
	now := time.Now()
	return s.DBConn.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    userID,
		FeedID:    feedID,
	})
}

func HandlerFeeds(s *app_state.AppState, cmd Command) error {
	feeds, err := s.DBConn.GetFeeds(context.Background())
	if err != nil {
		fmt.Printf("Error fetching feeds: %v\n", err)
		os.Exit(1)
	}

	for _, feed := range feeds {
		fmt.Printf("* %s (%s) added by %s\n", feed.Name, feed.Url, feed.UserName)
	}

	return nil
}

func HandlerBrowse(s *app_state.AppState, cmd Command, user database.User) error {
	limit := 2
	if len(cmd.Args) > 0 {
		parsedLimit, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("error parsing limit: %w", err)
		}
		limit = parsedLimit
	}

	posts, err := s.DBConn.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		fmt.Printf("Error fetching posts: %v\n", err)
		os.Exit(1)
	}

	for _, post := range posts {
		fmt.Printf("* %s (%s)\n", post.Title, post.Url)
	}

	return nil
}

func HandlerReset(s *app_state.AppState, cmd Command) error {
	err := s.DBConn.DeleteAllUsers(context.Background())
	if err != nil {
		fmt.Printf("Error resetting users: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("All users deleted successfully.")
	return nil
}
