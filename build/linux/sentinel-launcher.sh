#!/bin/sh
set -eu

bootstrap=/usr/lib/sentinel/sentinel-bootstrap.tar.gz
data_root=${XDG_DATA_HOME:-${HOME:?HOME must be set}/.local/share}
client_dir=$data_root/sentinel/app
client=$client_dir/sentinel

fail() {
    printf 'Sentinel: %s\n' "$*" >&2
    exit 1
}

# Package upgrades and reinstalls must preserve an initialized client.
if [ -f "$client" ] && [ -x "$client" ]; then
    exec "$client" "$@"
fi

mkdir -p "$client_dir" || fail "Cannot create client directory: $client_dir"
command -v flock >/dev/null 2>&1 || fail "Install util-linux to initialize the desktop client."
exec 9>"$client_dir/.bootstrap.lock"
flock -x 9 || fail "Cannot lock client directory: $client_dir"

# Another launcher may have initialized the client while we waited.
if [ ! -e "$client" ] && [ ! -L "$client" ]; then
    [ -r "$bootstrap" ] || fail "Bootstrap archive missing: $bootstrap. Reinstall the Sentinel package."
    entries=$(tar -tzf "$bootstrap") || fail "Cannot read bootstrap archive. Reinstall the Sentinel package."
    [ "$entries" = sentinel ] || fail "Invalid bootstrap archive: expected one executable named sentinel."

    stage=$(mktemp -d "$client_dir/.bootstrap-XXXXXX") || fail "Cannot create bootstrap staging directory."
    trap 'rm -rf "$stage"' 0
    trap 'exit 1' HUP INT TERM
    tar -xzf "$bootstrap" --no-same-owner -C "$stage" || fail "Cannot extract bootstrap archive. Reinstall the Sentinel package."
    [ -f "$stage/sentinel" ] && [ ! -L "$stage/sentinel" ] || fail "Bootstrap does not contain a regular executable."
    chmod 755 "$stage/sentinel" || fail "Cannot set client executable permissions."
    mv "$stage/sentinel" "$client" || fail "Cannot initialize client: $client"
    rm -rf "$stage"
    trap - 0 HUP INT TERM
fi

[ -f "$client" ] && [ -x "$client" ] || fail "Client is not executable: $client. Move it aside and launch Sentinel to restore the packaged client."

# Do not pass the bootstrap lock to the long-running application.
flock -u 9
exec 9>&-
exec "$client" "$@"
