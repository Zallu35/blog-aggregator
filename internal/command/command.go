package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Zallu35/blog-aggregator/internal/app_state"
	"github.com/Zallu35/blog-aggregator/internal/database"
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

	_, err := s.DBConn.GetUser(context.Background(), username)
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
