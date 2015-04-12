package main

import (
	"code.google.com/p/gcfg"
	"os"
)

// Configuration is Ourd's configuration
type Configuration struct {
	HTTP struct {
		Host string
	}
	App struct {
		Name   string
		APIKey string `gcfg:"api-key"`
	}
	DB struct {
		ImplName string `gcfg:"implementation"`
		Option   string
	}
	TokenStore struct {
		Path string `gcfg:"path"`
	} `gcfg:"token-store"`
	Subscription struct {
		Enabled bool
	}
	APNS struct {
		Gateway  string
		CertPath string `gcfg:"cert-path"`
		KeyPath  string `gcfg:"key-path"`
	}
	LOG struct {
		Level string
	}
}

// ReadFileInto reads a configuration from file specified by path
func ReadFileInto(config *Configuration, path string) error {
	if err := gcfg.ReadFileInto(config, path); err != nil {
		return err
	}
	if config.HTTP.Host == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}
		config.HTTP.Host = ":" + port
	}
	if config.DB.ImplName == "pq" && config.DB.Option == "" {
		config.DB.Option = os.Getenv("DATABASE_URL")
	}
	return nil
}