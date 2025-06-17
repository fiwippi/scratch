package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/fiwippi/halo"
)

func run(configPath string) error {
	if configPath == "" {
		return fmt.Errorf("config path not specified")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	var config halo.Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	// We need to omit the credentials for the log
	cleanConfig := config
	cleanConfig.Credentials = nil
	slog.Info("Loaded config", slog.Any("config", cleanConfig))

	s, err := halo.NewServer(config)
	if err != nil {
		return fmt.Errorf("create server: %s", err)
	}
	if err := s.ListenAndServe(); err != nil {
		return fmt.Errorf("listen and serve: %s", err)
	}

	return nil
}

func main() {
	configPath := flag.String("config", "", "Path to load config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		slog.Error("Run failed", slog.Any("err", err))
	}
}
