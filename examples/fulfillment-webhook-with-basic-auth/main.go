// Copyright 2018 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/kelseyhightower/dialogflow/fulfillment"
	"google.golang.org/api/dialogflow/v2beta1"
)

var (
	addr           string
	hashedPassword string
	username       string
)

type helloParameters struct {
	Name string `json:"name"`
}

func main() {
	flag.StringVar(&addr, "http", "127.0.0.1:80", "HTTP listen address")
	flag.StringVar(&hashedPassword, "hashed-password", "", "Basic auth hashed password")
	flag.StringVar(&username, "username", "", "Basic auth username")
	flag.Parse()

	fs := fulfillment.NewServer()
	fs.Addr = addr
	fs.BasicAuthHashedPassword = hashedPassword
	fs.BasicAuthUsername = username

	fs.Actions.Set("hello", hello)
	if err := fs.ListenAndServe(); err != nil {
		log.Println(err)
	}
}

func hello(q *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error) {
	var parameters helloParameters

	if err := json.Unmarshal(q.QueryResult.Parameters, &parameters); err != nil {
		return nil, err
	}

	response := &dialogflow.WebhookResponse{
		FulfillmentText: fmt.Sprintf("Hello %s!", parameters.Name),
	}
	return response, nil
}
