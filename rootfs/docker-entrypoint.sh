#!/bin/bash
set -e

if [ "$1" = "run-server" ]; then
    if ! dpkg -l linux-headers-$(uname -r) | tail -1 | grep -qE '^ii'; then
        apt-get update
        apt-get install --yes --no-install-recommends linux-headers-$(uname -r)
        apt-get clean && rm -rf /var/lib/apt/lists/*
    fi
    MODULE_VERSION=$(dpkg -l | grep 'wireguard-dkms' | awk '{ print $3 }' | awk -F '-' '{ print $1 }')
    dkms install wireguard/$MODULE_VERSION
    modprobe wireguard

    config=$(ls /etc/wireguard/*.conf | head -1)
    if ip a | grep -q $(basename $config | cut -f 1 -d '.'); then
        echo "Stopping existing interface"
        wg-quick down $config
    fi

    echo "Starting wireguard using $config"
    wg-quick up $config
    echo "Running config:"
    wg

    syncpeers() {
        echo 'Starting reconciliate process ...'
        # TODO: perform a diff to see if the system has revoked peers to remove
        # remove revoked peers
        for revoked in $(wgapp peers list $SERVER --filter=revoked); do
            echo "revoking peer $peer ..."
            wg set $config peer $peer remove
            break
        done

        # TODO: check if the current peers is equal to the active peers before adding

        # remove if the desired peer doesn't exists - garbage collector
        for cur in $(wg show $config peers); do
            exists=0
            for desired in $(wgapp peers list $SERVER --filter=active); do
                if [ "$cur" == "$desired" ]; then
                    exists=1
                    break
                fi
            done
            if [ "$exists" == "0" ]; then
                echo "removing peer $peer ..."
                wg set $config peer $peer remove
            fi
        done

        # create the desired peer if doesn't exists - reconcialition
        for desired in $(wgapp peers list $SERVER --filter=active); do
            exists=0
            for cur in $(wg show $config peers); do
                if [ "$cur" == "$desired" ]; then
                    exists=1
                    break
                fi
            done
            if [ "$exists" == "0" ]; then
                echo "adding peer $peer ..."
                wg set $config peer $peer
            fi
        done
    }

    shutdown() {
        echo "Stopping wireguard"
        wg-quick down $config
        rmmod wireguard
        echo "Uninstalling dkms module"
        dkms uninstall wireguard/$MODULE_VERSION
        exit 0
    }
    trap shutdown SIGINT SIGTERM SIGQUIT
    trap syncpeers SIGHUP

    sleep infinity &
    wait

elif [ "$1" = "gen-key" ]; then
    PRIVATE_KEY=$(wg genkey)
    PUBLIC_KEY=$(echo ${PRIVATE_KEY} | wg pubkey)
    echo "Private key: ${PRIVATE_KEY}"
    echo "Public key: ${PUBLIC_KEY}"
    exit 0
else
    exec $@
fi