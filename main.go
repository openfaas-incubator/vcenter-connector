package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openfaas-incubator/vcenter-connector/pkg/events"

	"github.com/openfaas-incubator/connector-sdk/types"
	"github.com/openfaas/faas-provider/auth"
	"github.com/openfaas/openfaas-cloud/sdk"
)

func main() {
	var gatewayURL string
	var vcenterURL string
	var vcUser string
	var vcPass string
	var vcUserSecret string
	var vcPasswordSecret string

	var insecure bool

	// TODO: add secrets management, verbosity level
	flag.StringVar(&gatewayURL, "gateway", "http://127.0.0.1:8080", "URL for OpenFaaS gateway")
	flag.StringVar(&vcenterURL, "vcenter", "http://127.0.0.1:8989/sdk", "URL for vCenter")
	flag.StringVar(&vcUser, "vc-user", "", "User to connect to vCenter")
	flag.StringVar(&vcPass, "vc-pass", "", "Password to connect to vCenter")

	flag.StringVar(&vcUserSecret, "vc-user-secret-name", "", "Secret file to use for username")
	flag.StringVar(&vcPasswordSecret, "vc-pass-secret-name", "", "Secret file to use for password")

	flag.BoolVar(&insecure, "insecure", false, "use an insecure connection to vCenter (default false)")
	flag.Parse()

	if len(vcenterURL) == 0 {
		log.Fatal("vcenterURL not provided")
	}

	if len(vcUserSecret) > 0 {
		val, err := sdk.ReadSecret(vcUserSecret)
		if err != nil {
			panic(err.Error())
		}
		vcUser = val
	}

	if len(vcPasswordSecret) > 0 {
		val, err := sdk.ReadSecret(vcPasswordSecret)
		if err != nil {
			panic(err.Error())
		}
		vcPass = val
	}

	vcenterClient, err := events.NewVCenterClient(context.Background(), vcUser, vcPass, vcenterURL, insecure)
	if err != nil {
		log.Fatalf("could not connect to vCenter: %v", err)
	}

	// OpenFaaS credentials for the connector
	var credentials *auth.BasicAuthCredentials
	if val, ok := os.LookupEnv("basic_auth"); ok && len(val) > 0 {
		if val == "true" || val == "1" {

			reader := auth.ReadBasicAuthFromDisk{}

			if val, ok := os.LookupEnv("secret_mount_path"); ok && len(val) > 0 {
				reader.SecretMountPath = os.Getenv("secret_mount_path")
			}

			res, err := reader.Read()
			if err != nil {
				log.Fatalf("could not read credentials: %v", err)
			}
			credentials = res
		}
	}

	// OpenFaaS connector SDK controller configuration
	ofconfig := types.ControllerConfig{
		GatewayURL:      gatewayURL,
		PrintResponse:   false,
		RebuildInterval: time.Second * 10,
		UpstreamTimeout: time.Second * 15,
	}

	// get OpenFaaS connector controller
	ofcontroller := types.NewController(credentials, &ofconfig)
	ofcontroller.BeginMapBuilder()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, os.Interrupt)
	go func() {
		s := <-sigCh
		log.Printf("got signal: %v, cleaning up...", s)
		cancel()
		// give subroutines some time to finish their work
		<-time.Tick(3 * time.Second)
		os.Exit(0)
	}()

	// blocks until eventStream returns
	err = events.Stream(ctx, vcenterClient.Client, ofcontroller)
	if err != nil {
		log.Fatalf("could not bind events: %v", err)
	}
}
