// Copyright 2018 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package fulfillment_test

import (
	"github.com/kelseyhightower/dialogflow/fulfillment"
	"google.golang.org/api/dialogflow/v2"
)

func Example() {
	fs := fulfillment.NewServer()
	fs.DisableBasicAuth = true

	fs.Actions.Set("helloworld", func(q *dialogflow.GoogleCloudDialogflowV2WebhookRequest) (*dialogflow.GoogleCloudDialogflowV2WebhookResponse, error) {
		response := &dialogflow.GoogleCloudDialogflowV2WebhookResponse{
			FulfillmentText: "Hello World!",
		}
		return response, nil
	})

	fs.ListenAndServe()
}
