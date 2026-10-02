# Ruijie Cloud connector

Base: `https://cloud.ruijienetworks.com` (partner access required).
Auth: API key/secret -> token (handled in `cloud.RuijieCloud`).
Status: REQUIRES_VENDOR_ACCESS. No endpoints fabricated; discovery,
device/AP/switch/gateway enumeration and webhook receiver are stubbed
pending documented credentials. Provide `api_key` + `api_secret` in
connector config, then run Connection Lab.
