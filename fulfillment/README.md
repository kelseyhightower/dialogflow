# fulfillment

## Example Usage

```
package main

import (
    "fmt"

    "google.golang.org/api/dialogflow/v2beta1"
    "github.com/kelseyhightower/dialogflow/fulfillment"
)

func main() {
    fs := fulfillment.NewServer()
    fs.Actions.Set("hello", hello)
    fs.ListenAndServe()
}

func hello(q *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error) {
    response := &dialogflow.Response{
        Speech: fmt.Sprintf("Hello %s", q.Result.Parameters["given-name"]),
    }
    return response, nil
}
```
