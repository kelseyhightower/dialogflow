# Cleanup

### DNS

```
gcloud dns record-sets transaction start --zone hightowerlabs
```

```
gcloud dns record-sets transaction remove --zone hightowerlabs \
  --name apiai-webhook.hightowerlabs.com \
  --ttl 30 \
  --type A ${APIAI_WEBHOOK_IP_ADDRESS}
```

```
gcloud dns record-sets transaction execute --zone hightowerlabs
```

### Firewall Rules

```
gcloud -q compute firewall-rules delete allow-apiai-fulfillment-webhook
```

### Compute Instances

```
gcloud -q compute instances delete apiai-fulfillment-webhook
```

### Compute Addresses

```
gcloud -q compute addresses delete apiai-fulfillment-webhook \
  --region $(gcloud config get-value compute/region)
```
