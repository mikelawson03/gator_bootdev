package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func (c *Config) SetUser() error {
	configPath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(c, "", "  ")

	if err = os.WriteFile(configPath, jsonBytes, 0644); err != nil {
		return err
	}

	return nil
}

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("User home directory not found")
	}

	configPath := home + "/" + configFileName

	return configPath, nil
}

func Read() (Config, error) {
	configPath, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	res, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("Error reading config file: %v", configPath)
	}

	var config Config

	if err := json.Unmarshal(res, &config); err != nil {
		return Config{}, fmt.Errorf("Error unmarshaling JSON")
	}

	return config, nil

}
