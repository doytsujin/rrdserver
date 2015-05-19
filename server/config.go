package rrdserver

import (
	"code.google.com/p/gcfg"
	"flag"
	"fmt"
	"github.com/rrdserver/rrdserver/log"
	"strings"
)

type Config struct {
	Server struct {
		Port     int
		Bind     string
		User     string
		Password string
	}

	Metrics struct {
		DataDir string
	}
}

func NewConfig() Config {
	var configFile string
	var argPort int
	var argBind string

	flag.StringVar(&configFile, "config", "/etc/rrdserv.conf", "Path to config file")
	flag.StringVar(&configFile, "c", "/etc/rrdserv.conf", "Path to config file (shorthand)")

	flag.IntVar(&argPort, "port", 8085, "Port number to use for connection")
	flag.StringVar(&argBind, "bind", "0.0.0.0", "The address a listening server should bind to.")

	flag.Parse()

	var cfg Config

	if err := gcfg.ReadFileInto(&cfg, configFile); err != nil {
		fmt.Printf("Can't read config: %v\n", err)
		log.Fatal("Can't read config: %v", err)
	}

	if flagIsSet("port") {
		cfg.Server.Port = argPort
	}

	if flagIsSet("bind") {
		cfg.Server.Bind = argBind
	}

	if cfg.Metrics.DataDir == "" {
		fmt.Printf("Config error. DataDir isn't set.\n")
		log.Fatal("Config error. DataDir isn't set.")
	}

	if !strings.HasSuffix(cfg.Metrics.DataDir, "/") {
		cfg.Metrics.DataDir += "/"
	}

	return cfg
}

func flagIsSet(name string) bool {
	res := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			res = true
		}
	})
	return res
}
