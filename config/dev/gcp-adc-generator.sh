#!/bin/bash

set -e

e() {
  echo "ERROR: ${BASH_COMMAND}" >&2
}

trap e ERR

ADC_PATH="${GOOGLE_APPLICATION_CREDENTIALS:-$HOME/.config/gcloud/application_default_credentials.json}"
kubectl create secret generic gcp-adc --from-file "key.json=$ADC_PATH" --dry-run=client -o yaml > /dev/stdout