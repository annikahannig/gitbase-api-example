package main

import (
	"flag"
)

type HttpConfig struct {
	Listen string
}

type Config struct {
	Http     *HttpConfig
	RepoPath string
}

func parseFlags() *Config {
	repo := flag.String("path", "", "Path to files")
	httpListen := flag.String("listen", ":8042", "HTTP Listen Port")

	flag.Parse()

	httpConfig := &HttpConfig{
		Listen: *httpListen,
	}

	config := &Config{
		Http:     httpConfig,
		RepoPath: *repo,
	}

	return config
}
