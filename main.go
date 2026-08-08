package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/Zallu35/blog-aggregator/internal/app_state"
	"github.com/Zallu35/blog-aggregator/internal/command"
	"github.com/Zallu35/blog-aggregator/internal/config"
	"github.com/Zallu35/blog-aggregator/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}

	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		fmt.Printf("Error connecting to database")
		os.Exit(1)
	}
	dbQueries := database.New(db)

	state := &app_state.AppState{
		Config: &cfg,
		DBConn: dbQueries,
	}

	cmds := &command.Commands{}
	cmds.Register("login", command.HandlerLogin)
	cmds.Register("register", command.HandlerRegister)
	cmds.Register("reset", command.HandlerReset)
	cmds.Register("users", command.HandlerUsers)
	cmds.Register("agg", command.HandlerAgg)
	cmds.Register("addfeed", command.MiddlewareLoggedIn(command.HandlerAddFeed))
	cmds.Register("feeds", command.HandlerFeeds)
	cmds.Register("follow", command.MiddlewareLoggedIn(command.HandlerFollow))
	cmds.Register("following", command.MiddlewareLoggedIn(command.HandlerFollowing))
	cmds.Register("unfollow", command.MiddlewareLoggedIn(command.HandlerUnfollow))

	// Get command-line arguments
	if len(os.Args) < 2 {
		log.Fatal("Please provide a command.")
	}

	commandName := os.Args[1]
	commandArgs := os.Args[2:]

	cmd := command.Command{
		Name: commandName,
		Args: commandArgs,
	}

	err = cmds.Run(state, cmd)
	if err != nil {
		fmt.Printf("Error running command \"%s\": %v\n", cmd.Name, err)
		os.Exit(1)
	}
}
