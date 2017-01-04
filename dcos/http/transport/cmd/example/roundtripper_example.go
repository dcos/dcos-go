package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dcos/dcos-go/jwt/transport"
)

var (
	flagURL       = flag.String("url", "", "URL to query")
	flagIAMConfig = flag.String("iam-config", "", "Path to IAM config")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: %s -url http://127.0.0.1/system/health/v1 -iam-config /run/dcos/etc/3dt/master_service_account.json\n\n", os.Args[0])
	}
	flag.Parse()
	if *flagURL == "" || *flagIAMConfig == "" {
		flag.Usage()
		os.Exit(1)
	}

	c := &http.Client{}
	rt, err := transport.NewRoundTripper(c.Transport,
		transport.OptionReadIAMConfig(*flagIAMConfig),
		transport.OptionTokenExpire(time.Duration(time.Second*2)))
	if err != nil {
		log.Fatal(err)
	}
	c.Transport = rt

	req, _ := http.NewRequest("GET", *flagURL, nil)

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
