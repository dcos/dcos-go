package main

import (
	_ "github.com/dcos/dcos-go/dcos-log/api"
	_ "github.com/dcos/dcos-go/dcos-log/router"
	_ "github.com/dcos/dcos-go/dcos-log/journal/reader"
	"github.com/dcos/dcos-go/dcos-log/config"
	"os"
	"github.com/Sirupsen/logrus"
)

func main() {
	cfg, err := config.NewConfig(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("%+v", cfg)
}
