package main

import (
	"flag"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/openark/golib/log"

	"op-agent/app"
	"op-agent/config"
)

// main is the application's entry point. It will either spawn a CLI or HTTP itnerfaces.
func main() {
	configFile := flag.String("config", "", "config file name")
	verbose := flag.Bool("verbose", false, "verbose")
	debug := flag.Bool("debug", false, "debug mode (very verbose)")
	stack := flag.Bool("stack", false, "add stack trace upon error")
	version := flag.Bool("version", false, "app version")
	flag.Parse()

	log.SetLevel(log.ERROR)
	if *verbose {
		log.SetLevel(log.INFO)
	}
	if *debug {
		log.SetLevel(log.DEBUG)
	}
	if *stack {
		log.SetPrintStackTrace(*stack)
	}

	appVersion := config.NewAppVersion()

	if *version {
		fmt.Println(appVersion)
		return
	}

	startText := "starting op-agent"
	if appVersion != "" {
		startText += ", version: " + appVersion
	}
	log.Info(startText)
	if len(*configFile) > 0 {
		config.ForceRead(*configFile)
	} else {
		config.Read("/etc/op-agent.conf.json",
			"conf/op-agent.conf.json",
			"op-agent.conf.json")
	}
	if config.Config.Debug {
		log.SetLevel(log.DEBUG)
	}
	config.MarkConfigurationLoaded()
	app.Http("op-manager")
}
