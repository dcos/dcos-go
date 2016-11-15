package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/dcos/dcos-go/dcos/nodeutil"
	"github.com/dcos/dcos-go/jwt/transport"
)

var flagIAMConfig = flag.String("iam-config", "", "Path to IAM config")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: %s -iam-config /run/dcos/etc/3dt/master_service_account.json\n\n", os.Args[0])
	}
	flag.Parse()
	if *flagIAMConfig == "" {
		flag.Usage()
		os.Exit(1)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	rt, err := transport.NewRoundTripper(client.Transport, transport.OptionReadIAMConfig(*flagIAMConfig))
	if err != nil {
		panic(err)
	}
	client.Transport = rt

	d, err := nodeutil.NewNodeInfo(client)
	if err != nil {
		panic(err)
	}

	ip, err := d.DetectIP()
	if err != nil {
		panic(err)
	}
	fmt.Printf("IP=%s\n", ip.String())

	r, err := d.Role()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Node's role %s\n", r)

	leader, err := d.IsLeader()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Leader %v\n", leader)

	mesosID, err := d.MesosID(nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("MesosID %s\n", mesosID)
}
