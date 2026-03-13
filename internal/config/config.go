package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func Read() (Config, error) {
	var configStruct = Config{}
	path, e := getConfigFilePath()
	if e != nil {
		return Config{}, fmt.Errorf("Error getting file path: %v", e)
	}
	cfgFile, er := os.ReadFile(path)
	if er != nil {
		return Config{}, fmt.Errorf("Error reading config file %v", er)
	}
	if err := json.Unmarshal(cfgFile, &configStruct); err != nil {
		return Config{}, fmt.Errorf("Error unmarshalling config file: %v", err)
	}
	return configStruct, nil
}

func getConfigFilePath() (string, error) {
	path, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Error fetching home directory path: %v", err)
	}
	return path + "/.gatorconfig.json", nil
}

func (c *Config) SetUser(userName string) error {
	c.CurrentUserName = userName
	jsonData, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("Error marshalling SetUser data: %v", err)
	}
	path, e := getConfigFilePath()
	if e != nil {
		return fmt.Errorf("Error getting ConfigFilePath: %v", e)
	}
	er := os.WriteFile(path, jsonData, 0644)
	if er != nil {
		return fmt.Errorf("Error writing to file: %v", er)
	}
	return nil
}
