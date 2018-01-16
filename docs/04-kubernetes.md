# Kubernetes

In this section you will create a Kubernetes cluster and deploy the `apiai-fulfillment-webhook`
container to it.

## Kubernetes Cluster

Create a Kubernetes cluster:

```
gcloud container clusters create apiai-fulfillment-webhook \
  --cluster-version 1.7.5
```

Store the `basic-auth.json` basic auth file in a Kubernetes secret:

```
kubectl create secret generic apiai-fulfillment-webhook \
  --from-file basic-auth.json
```

Set the domain and bucket configuration parameters:

```
kubectl create configmap apiai-fulfillment-webhook \
  --from-literal domain=apiai-fulfillment-webhook.hightowerlabs.com \
  --from-literal bucket=hightowerlabs
```

### The apiai-fulfillment-webhook Deployment

Create the `apiai-fulfillment-webhook` deployment:

```
kubectl create -f \
  https://storage.googleapis.com/apiai-fulfillment-webhook/apiai-fulfillment-webhook.yaml
```

Retrieve the `apiai-fulfillment-webhook` IP address:

```
APIAI_FULFILLMENT_WEBHOOK_IP_ADDRESS=$(gcloud compute addresses describe \
    apiai-fulfillment-webhook \
    --region $(gcloud config get-value compute/region) \
    --format 'value(address)')
```

Create the `apiai-fulfillment-webhook` service using the `apiai-fulfillment-webhook` static IP address:

```
cat <<EOF | kubectl apply -f -
kind: Service
apiVersion: v1
metadata:
  name: apiai-fulfillment-webhook
spec:
  selector:
    app: apiai-fulfillment-webhook
  ports:
    - protocol: TCP
      port: 443
      targetPort: 443
  loadBalancerIP: ${APIAI_FULFILLMENT_WEBHOOK_IP_ADDRESS}
  type: LoadBalancer
EOF
```
