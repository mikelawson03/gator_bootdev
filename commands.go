package main

import (
	"context"
	"fmt"
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
