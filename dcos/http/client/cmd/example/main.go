package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dcos/dcos-go/dcos/http/client"
)

var (
	url        = flag.String("url", "", "URL to query")
	iamConfig  = flag.String("iam-config", "", "Path to IAM config")
	caCertPath = flag.String("ca-cert-path", "", "Path to CA certificate")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: %s -url http://127.0.0.1/system/health/v1 -ca-cert-path /run/dcos/etc/ca_cert.pem -iam-config /run/dcos/etc/3dt/master_service_account.json\n\n", os.Args[0])
	}
	flag.Parse()
	if *url == "" || *iamConfig == "" || *caCertPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	c, err := client.NewDCOSClient(*caCertPath, *iamConfig)
	if err != nil {
		panic(err)
	}

	req, _ := http.NewRequest("GET", *url, nil)

	for {

		resp, err := c.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("Expecting return code 200. Got %d", resp.StatusCode)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(body))
		resp.Body.Close()
		time.Sleep(time.Second)
	}
}
