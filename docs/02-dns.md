# DNS

## Static IP Address

Allocate a static IP address:

```
gcloud compute addresses create apiai-fulfillment-webhook \
  --region $(gcloud config get-value compute/region)
```

```
APIAI_FULFILLMENT_WEBHOOK_IP_ADDRESS=$(gcloud compute addresses describe \
    apiai-fulfillment-webhook \
    --region $(gcloud config get-value compute/region) \
    --format 'value(address)')
```

## DNS Record

```
DNS_ZONE="hightowerlabs"
```

```
DOMAIN="hightowerlabs.com"
```

Create a DNS record:

```
gcloud dns record-sets transaction start --zone=${DNS_ZONE}
```

```
gcloud dns record-sets transaction add \
  --zone ${DNS_ZONE} \
  --name "apiai-fulfillment-webhook.${DOMAIN}." \
  --ttl 30 \
  --type A ${APIAI_FULFILLMENT_WEBHOOK_IP_ADDRESS}
```

```
gcloud dns record-sets transaction describe --zone ${DNS_ZONE}
```

```
gcloud dns record-sets transaction execute --zone ${DNS_ZONE}
```

```
gcloud dns record-sets list --zone ${DNS_ZONE}
```

## Firewall Rules

```
gcloud compute firewall-rules create allow-apiai-fulfillment-webhook \
  --allow tcp:443,tcp:8443,tcp:9090
```
