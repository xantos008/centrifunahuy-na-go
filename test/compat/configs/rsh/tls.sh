#!/bin/sh
# Generate the rsh listener's test certificates before its config is mounted.
set -eu
tls_dir="$(dirname "$0")/tls"
if [ -s "$tls_dir/ca.crt" ] && [ -s "$tls_dir/server.key" ] && [ -s "$tls_dir/server.crt" ] &&
    openssl x509 -in "$tls_dir/server.crt" -noout -checkend 3600 >/dev/null &&
    openssl verify -CAfile "$tls_dir/ca.crt" -purpose sslserver -verify_ip 127.0.0.1 "$tls_dir/server.crt" >/dev/null &&
    [ "$(openssl pkey -in "$tls_dir/server.key" -pubout)" = "$(openssl x509 -in "$tls_dir/server.crt" -noout -pubkey)" ]; then
    exit 0
fi

umask 077
work=$(mktemp -d)
trap 'rm -f "$work/ca.key" "$work/ca.crt" "$work/server.key" "$work/server.csr" "$work/server.crt" "$work/extensions"; rmdir "$work"' EXIT
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
    -keyout "$work/ca.key" -out "$work/ca.crt" -days 30 -subj '/CN=compat rsh CA' \
    -addext 'basicConstraints=critical,CA:TRUE' -addext 'keyUsage=critical,keyCertSign'
openssl req -new -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
    -keyout "$work/server.key" -out "$work/server.csr" -subj '/CN=localhost'
cat >"$work/extensions" <<'EOF'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=serverAuth
subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1
EOF
openssl x509 -req -in "$work/server.csr" -CA "$work/ca.crt" -CAkey "$work/ca.key" \
    -set_serial 2 -days 7 -extfile "$work/extensions" -out "$work/server.crt"
# Both server images must be able to read the test leaf key under their own UID.
# The CA private key stays in the private staging directory and is discarded.
install -d -m 755 "$tls_dir"
rm -f "$tls_dir/server.crt" "$tls_dir/server.key" "$tls_dir/ca.crt"
install -m 444 "$work/server.crt" "$work/server.key" "$work/ca.crt" "$tls_dir"
