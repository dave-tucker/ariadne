#!/bin/bash

set -e

kubectl exec -it -n network-researcher \
    $(kubectl get pods -n network-researcher -o json | jq -r '.items[0].metadata.name') \
    -c network-researcher \
    -- uv run network-researcher -i
