# systemd

## Compute Instance

```
APIAI_FULFILLMENT_WEBHOOK_IP_ADDRESS=$(gcloud compute addresses describe \
    apiai-fulfillment-webhook \
    --region $(gcloud config get-value compute/region) \
    --format 'value(address)')
```

Create a compute instance and assign it the `apiai-fulfillment-webhook` static IP address:

```
gcloud compute instances create apiai-fulfillment-webhook \
  --address ${APIAI_FULFILLMENT_WEBHOOK_IP_ADDRESS} \
  --async \
  --image-family ubuntu-1704 \
  --image-project ubuntu-os-cloud \
  --machine-type f1-micro \
  --scopes compute-rw,storage-rw,logging-write,monitoring
```

## Deploy

```
gcloud compute ssh apiai-fulfillment-webhook
```

```
wget -q --show-progress --https-only --timestamping \
  https://storage.googleapis.com/apiai-fulfillment-webhook/apiai-fulfillment-webhook \
  https://storage.googleapis.com/apiai-fulfillment-webhook/apiai-fulfillment-webhook.service
```

```
sudo mkdir -p /etc/apiai-fulfillment-webhook/
```

```
chmod +x apiai-fulfillment-webhook
```

```
sudo mv apiai-fulfillment-webhook /usr/local/bin
```

```
cat > environment <<EOF
BASIC_AUTH_FILE=/etc/apiai-fulfillment-webhook/basic-auth.json
BUCKET=hightowerlabs
DOMAIN=hightowerlabs.com.
EOF
```

```
sudo mv environment /etc/apiai-fulfillment-webhook/
```

```
sudo mv apiai-fulfillment-webhook.service /etc/systemd/system/
```

```
sudo systemctl daemon-reload
```

```
sudo systemctl enable apiai-fulfillment-webhook
```

```
sudo systemctl start apiai-fulfillment-webhook
```
