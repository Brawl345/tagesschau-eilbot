package storage

import (
	"encoding/json"
	"errors"
	"os"
)

const fileName string = "config.json"

type Config struct {
	Token       string  `json:"token"`
	LastEntry   string  `json:"last_entry"`
	Subscribers []int64 `json:"subscribers"`
	Debug       bool    `json:"debug"`
}

func (config *Config) Load() error {
	configFile, err := os.ReadFile(fileName)

	if err != nil {
		return err
	}

	err = json.Unmarshal(configFile, &config)
	return nil
}

func (config *Config) Save() error {
	file, err := json.MarshalIndent(config, "", "  ")

	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, file, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (config *Config) AddSubscriber(chatId int64) error {
	isSubscriber, _ := config.isSubscriber(chatId)
	if isSubscriber {
		return errors.New("chatId is already a subscriber")
	}
	config.Subscribers = append(config.Subscribers, chatId)

	return nil
}

func (config *Config) RemoveSubscriber(chatId int64) error {
	isSubscriber, index := config.isSubscriber(chatId)
	if !isSubscriber {
		return errors.New("chatId is not a subscriber")
	}
	config.Subscribers[index] = config.Subscribers[len(config.Subscribers)-1]
	config.Subscribers = config.Subscribers[:len(config.Subscribers)-1]

	return nil
}

func (config *Config) isSubscriber(chatId int64) (bool, int) {
	for i, v := range config.Subscribers {
		if v == chatId {
			return true, i
		}
	}

	return false, -1
}
