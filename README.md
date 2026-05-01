# Gator RSS

Gator is a CLI aggregator that allows users to add RSS feeds, follow feeds from others, aggregate feeds, and browse feeds.

## Dependencies
Gator requires PostgreSQL and Golang

To install PostgreSQL

```
sudo apt update
sudo apt install postgresql postgresql-contrib
```
To install Go:

```
sudo apt-get update
sudo apt-get -y install golang-go
```

## Installation

Install package using go install

```bash
go install github.com/mikelawson03/gator_bootdev@latest
```

## Setup
Create file at `~/.gatorconfig.json` with the following contents:

```
{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable"
}
```

## Usage

```bash

# register new user
gator_bootdev register <username>

# user login
gator_bootdev login <username>

# list users
gator_bootdev users

# add new RSS feed
gator_bootdev addfeed <feed name> <url>

# aggregate news every <duration> amount of time (e.g., 5m, 10s, etc)
gator_bootdev agg <duration>

# see list of all feeds
gator_bootdev feeds

# follow a feed
gator_bootdev follow <url>

# unfollow a feed
gator_bootdev unfollow <url>

# see list of feeds followed by current user
gator_bootdev following

# browse feeds followed by users; can use optional limit to return that number of results; default limit = 2
gator_bootdev browse (opt.) <limit>

# reset all user data
gator_bootdev reset
```

## License

[MIT](https://choosealicense.com/licenses/mit/)