package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Zallu35/blog-aggregator/internal/app_state"
	"github.com/Zallu35/blog-aggregator/internal/database"
	"github.com/Zallu35/blog-aggregator/internal/rss"
	"github.com/google/uuid"
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
	feed, err := rss.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return fmt.Errorf("error fetching feed: %w", err)
	}

	fmt.Printf("%+v\n", feed)
	return nil
}

func HandlerAddFeed(s *app_state.AppState, cmd Command) error {
	if len(cmd.Args) < 2 {
		return errors.New("the addfeed handler expects two arguments: the feed name and url")
	}

	name := cmd.Args[0]
	url := cmd.Args[1]

	user, err := s.DBConn.GetUserByName(context.Background(), s.Config.CurrentUserName)
	if err != nil {
		fmt.Printf("Error fetching current user '%s': %v\n", s.Config.CurrentUserName, err)
		os.Exit(1)
	}

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
