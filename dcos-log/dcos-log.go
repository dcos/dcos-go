package main

import (
	"github.com/Sirupsen/logrus"
	_ "github.com/dcos/dcos-go/dcos-log/api"
	"github.com/dcos/dcos-go/dcos-log/config"
	_ "github.com/dcos/dcos-go/dcos-log/journal/reader"
	_ "github.com/dcos/dcos-go/dcos-log/router"
	"os"
)

func main() {
	cfg, err := config.NewConfig(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("%+v", cfg)
}
