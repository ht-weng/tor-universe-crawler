#!/bin/sh

USER_CFG=/tmp/torrc-user
DEFAULT_CFG=/torrc

has_custom_setting() {
    if [ -z "$1" ]; then
        echo 0
    else
        echo 1
    fi
}

save_custom_setting() {
    rm "$1" -f

    echo "$2" > "$1"
}

file_enabled() {
    if [ -z "$(which $1)" ]; then
        echo "missing"
    else
        echo "available"
    fi
}

echo
echo "    Rui NI's Dockerilzed Tor Proxy"
echo
echo "    For protecting your privacy, not hiding your crime :)"
echo
echo "    ranqus@gmail.com https://github.com/niruix/docker-tor"
echo

if [ $(file_enabled tor) != "available" ]; then
    echo "[ERR] Sorry, we've failed detecting Tor. Are you sure it has been installed?"
    echo "      Quitting ..."
    echo

    return
fi

echo "[INF] Starting Tor, version: $(tor --version) ..."
echo "      - obfs:        $(file_enabled obfs4proxy)"
echo "      - meek-server: $(file_enabled meek-server)"
echo "      - meek-client: $(file_enabled meek-client)"

if [ $(has_custom_setting "$TOR_CUSTOM_CONFIGURATION") -gt 0 ]; then
    echo "[INF] Custom setting detected, replacing current configuration file $USER_CFG ..."

    save_custom_setting "$USER_CFG" "$TOR_CUSTOM_CONFIGURATION"
fi

if [ $(file_enabled update-ca-certificates) == "available" ]; then
    echo "[INF] Trying to update system CA certificates ..."

    update-ca-certificates
fi

if [ -f "$USER_CFG" ]; then
    echo

    tor --defaults-torrc $DEFAULT_CFG -f $USER_CFG $@
else
    echo "[INF] No custom settings, use default from the Docker Image ..."
    echo

    tor --defaults-torrc $DEFAULT_CFG $@
fi
