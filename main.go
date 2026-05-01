package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/mikelawson03/gator_bootdev/internal/config"
	"github.com/mikelawson03/gator_bootdev/internal/database"
)

type state struct {
	db     *database.Queries
	cfg    *config.Config
	client *http.Client
}

func main() {
	// get config from file
	cfg, err := config.Read()
	if err != nil {
		fmt.Println(err)
	}

	// open connection and gather queries
	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		fmt.Println(fmt.Errorf("Unable to connect to database"))
		os.Exit(1)
	}
	dbQueries := database.New(db)

	//initialize client
	c := &http.Client{}

	// set current state
	s := &state{
		db:     dbQueries,
		cfg:    &cfg,
		client: c,
	}

	// create commands struct
	cmds := commands{
		cmds: make(map[string]func(*state, command) error),
	}

	// register new commands in struct
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))

	input := os.Args
	if len(input) < 2 {
		fmt.Println(fmt.Errorf("Please provide command name. Syntax: go run . <command>"))
		os.Exit(1)
	}

	cmd := command{
		name: input[1],
		args: input[2:],
	}

	if err = cmds.run(s, cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
