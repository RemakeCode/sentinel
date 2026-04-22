#!/usr/bin/env bash
CLI_LOCATION="$(pwd)/cli"
echo "Building plugin in $(pwd)"

# Rootless Docker usage requires a custom build of the CLI until the merged fix has been released in stable.
# PR: https://github.com/SteamDeckHomebrew/cli/pull/6
# Fixed code: https://github.com/XanderXAJ/decky-cli/tree/fix-rootless-bind-mount-permission
RUST_LOG=DEBUG $CLI_LOCATION/decky plugin build $(pwd)
