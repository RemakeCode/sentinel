#!/bin/sh

# RPM may run the old package's postremove after installing its replacement.
case "${1:-}" in
    upgrade|1) exit 0 ;;
esac

rm -f /var/local/sentinel-install-*
