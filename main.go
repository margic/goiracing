package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/margic/sim/driver-go/iracing"
	"go.uber.org/zap"
)

func main() {
	logger := newLogger()
	defer logger.Sync()
	client := iracing.NewClient(logger)

	err := client.Open()
	if err != nil {
		logger.Error("Error opening iracing client",
			zap.Error(err))
		os.Exit(1)
	}
	defer client.Close()

	// start a time loop
	go func() {
		for now := range time.Tick(1000 * time.Millisecond) {
			fmt.Println(now)
		}
	}()
}

func newLogger() *zap.Logger {
	rawJSON := []byte(`{
		"level": "debug",
		"encoding": "json",
		"outputPaths": ["stdout"],
		"errorOutputPaths": ["stderr"],
		"initialFields": {},
		"encoderConfig": {
		  "messageKey": "message",
		  "levelKey": "level",
		  "levelEncoder": "lowercase"
		}
	  }`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return logger
}
