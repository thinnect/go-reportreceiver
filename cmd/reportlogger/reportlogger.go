// Author  Raido Pahtma
// License MIT

// reportlogger executable.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/proactivity-lab/go-loggers"
	"github.com/proactivity-lab/go-moteconnection"
	"github.com/thinnect/go-reportreceiver"
)

const ApplicationVersionMajor = 0
const ApplicationVersionMinor = 4
const ApplicationVersionPatch = 0

var ApplicationBuildDate string
var ApplicationBuildDistro string

type Options struct {
	Positional struct {
		ConnectionString string `description:"Connectionstring sf@HOST:PORT or serial@PORT:BAUD"`
	} `positional-args:"yes"`

	Output string `short:"o" long:"output" default:"reports.txt" description:"Reports output file"`

	Address moteconnection.AMAddr  `short:"a" long:"address" default:"0001" description:"Local address (hex)"`
	Group   moteconnection.AMGroup `short:"g" long:"group"   default:"22"   description:"Packet AM Group (hex)"`

	Debug       []bool `short:"D" long:"debug"   description:"Debug mode, print raw packets"`
	ShowVersion func() `short:"V" long:"version" description:"Show application version"`
}

func mainfunction() int {

	var opts Options
	opts.ShowVersion = func() {
		if ApplicationBuildDate == "" {
			ApplicationBuildDate = "YYYY-mm-dd_HH:MM:SS"
		}
		if ApplicationBuildDistro == "" {
			ApplicationBuildDistro = "unknown"
		}
		fmt.Printf("reportlogger %d.%d.%d (%s %s)\n", ApplicationVersionMajor, ApplicationVersionMinor, ApplicationVersionPatch, ApplicationBuildDate, ApplicationBuildDistro)
		os.Exit(0)
	}

	_, err := flags.Parse(&opts)
	if err != nil {
		flagserr := err.(*flags.Error)
		if flagserr.Type != flags.ErrHelp {
			if len(opts.Debug) > 0 {
				fmt.Printf("Argument parser error: %s\n", err)
			}
			return 1
		}
		return 0
	}

	fmt.Println("Program started")

	conn, _, err := moteconnection.CreateConnection(opts.Positional.ConnectionString)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return 1
	}

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, os.Kill)

	rl := reportreceiver.NewReportReceiver(conn, opts.Address, opts.Group)
	rfw, _ := reportreceiver.NewReportFileWriter(opts.Output)
	rl.SetOutput(rfw)

	logger := loggers.BasicLogSetup(len(opts.Debug))
	if len(opts.Debug) > 0 {
		conn.SetLoggers(logger)
	}
	rl.SetLoggers(logger)

	conn.Autoconnect(30 * time.Second)

	go rl.Run()
	go rl.RunResetResender()
	go rl.RunMissingFragmentResender()
	for interrupted := false; interrupted == false; {
		select {
		case sig := <-signals:
			signal.Stop(signals)
			logger.Debug.Printf("signal %s\n", sig)
			conn.Disconnect()
			interrupted = true
			// also stop rl?
		}
	}

	time.Sleep(100 * time.Millisecond)
	return 0
}

func main() {
	os.Exit(mainfunction())
}
