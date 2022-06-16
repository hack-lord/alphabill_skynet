#!/bin/sh

readonly FAUCET_DIR=/faucet/.alphabill

# Output all executed lines
set -x

# Delete old wallet
rm -rf ${FAUCET_DIR}/wallet

# Create wallet
/local/alphabill wallet create --home ${FAUCET_DIR}

# Get public key
PUBKEY=$(/local/alphabill wallet get-pubkey --home ${FAUCET_DIR} | grep '0x')

# Transfer initial bill to new wallet
/local/alphabill-spend-initial-bill \
    --pubkey ${PUBKEY} \
    --alphabill-uri [[ .gt_environment ]]-ab-money-app.service.[[ .gt_domain ]]:[[ .ab_partition_api_port ]] \
    --bill-id 1 \
    --bill-value 1000000 \
    --timeout 100

# Publish new wallet public key
readonly CONSUL_URL="[[ .consul_nomad_url ]]"
readonly TOKEN="[[ .consul_nomad_faucet_token ]]"
readonly KEY_NAME="[[ .gt_environment ]]/faucet-pubkey"

readonly TIMESTAMP=$(date +%s)

curl \
    --request PUT \
    --header "X-Consul-Token: ${TOKEN}" \
    --data "${PUBKEY}" \
    ${CONSUL_URL}/v1/kv/${KEY_NAME}
