// Copyright 2018 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package fulfillment_test

import (
	"fmt"

	"google.golang.org/api/dialogflow/v2beta1"
	"github.com/kelseyhightower/dialogflow/fulfillment"
)

func Example() {
	fs := fulfillment.NewServer()

	fs.Actions.Set("hello", func(q *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error) {
		response := &dialogflow.WebhookResponse{
			Speech: fmt.Sprintf("Hello %s!", q.Result.Parameters["name"]),
		}
		return response, nil
	})

	fs.ListenAndServe()
}
