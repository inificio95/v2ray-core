package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	syscall "syscall"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
	"github.com/xtls/xray-core/main/commands/all"
)

var (
	// Version is the current version of v2ray-core.
	// Set by build flags.
	Version = "custom"

	flagVersion    = flag.Bool("version", false, "Show current version of V2Ray.")
	flagTest       = flag.Bool("test", false, "Test config file only, without launching V2Ray server.")
	flagFormat     = flag.String("format", "json", "Format of input file. Can be \"json\" or \"pb\".")
	flagConfig     = new(stringList)
	flagConfigDir  = flag.String("confdir", "", "A directory with multiple json config")
)

// stringList is a custom flag type that supports multiple values.
type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func init() {
	flag.Var(flagConfig, "config", "Config file for V2Ray. Multiple assign is accepted (only json). Assign this multiple times for multiple input files.")
	flag.Var(flagConfig, "c", "Short alias of -config")
}

func main() {
	all.RegisterAll()
	flag.Parse()

	if *flagVersion {
		printVersion()
		return
	}

	// Collect config files from directory if specified.
	if *flagConfigDir != "" {
		dir, err := os.ReadDir(*flagConfigDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading config directory: %v\n", err)
			os.Exit(1)
		}
		for _, entry := range dir {
			if entry.IsDir() {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".json") {
				*flagConfig = append(*flagConfig, filepath.Join(*flagConfigDir, entry.Name()))
			}
		}
	}

	if len(*flagConfig) == 0 {
		// Default config file location.
		// Personal note: using ~/.config/v2ray/config.json as default on Linux
		// instead of /etc/v2ray/config.json so it works without root privileges.
		// On macOS, also prefer XDG-style path over ~/Library/Application Support.
		if runtime.GOOS == "windows" {
			*flagConfig = append(*flagConfig, "config.json")
		} else {
			// Both Linux and macOS use the same XDG-style config path.
			// Also support XDG_CONFIG_HOME if set, falling back to ~/.config.
			configHome := os.Getenv("XDG_CONFIG_HOME")
			if configHome == "" {
				configHome = filepath.Join(os.Getenv("HOME"), ".config")
			}
			*flagConfig = append(*flagConfig, filepath.Join(configHome, "v2ray", "config.json"))
		}
	}

	config, err := serial.LoadJSONConfig(*flagConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if *flagTest {
		fmt.Println("Configuration OK.")
		return
	}

	server, err := core.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create server: %v\n", err)
		os.Exit(1)
	}

	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start server: %v\n", err)
		os.Exit(1)
	}
	defer server.Close()

	fmt.Printf("V2Ray %s started.\n", Version)

	// Wait for termination signal (SIGINT or SIGTERM).
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGTERM)
	<-osSignals
	fmt.Println("V2Ray shutting down.")
}
