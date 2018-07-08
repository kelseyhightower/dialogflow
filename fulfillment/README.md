# fulfillment

## Example Usage

```Go
package main

import (
    "fmt"

    "github.com/kelseyhightower/dialogflow/fulfillment"
    "google.golang.org/api/dialogflow/v2"
)

func main() {
    fs := fulfillment.NewServer()
    fs.Actions.Set("helloworld", helloworld)
    fs.DisableBasicAuth = true
    fs.ListenAndServe()
}

func helloworld(q *dialogflow.GoogleCloudDialogflowV2WebhookRequest) (*dialogflow.GoogleCloudDialogflowV2WebhookResponse, error) {
    response := &dialogflow.GoogleCloudDialogflowV2WebhookResponse{
        FulfillmentText: "Hello World!",
    }
    return response, nil
}
```
