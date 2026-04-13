#!/bin/bash

# Stop any running instances of sentinel before removal.
if pgrep -x "sentinel" >/dev/null 2>&1; then
  echo "Found running sentinel processes. Stopping them..."
  pkill -x "sentinel"
  
  # Wait up to 5 seconds for processes to exit
  count=0
  while pgrep -x "sentinel" >/dev/null 2>&1 && [ $count -lt 5 ]; do
    sleep 1
    count=$((count + 1))
  done
  
  # Force kill if still running
  if pgrep -x "sentinel" >/dev/null 2>&1; then
    echo "Warning: sentinel processes did not exit gracefully. Force killing..."
    pkill -9 -x "sentinel"
  fi
fi


# Remove autostart entry on uninstall
AUTOSTART_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/autostart"
AUTOSTART_FILE="$AUTOSTART_DIR/sentinel-autostart.desktop"

if [ -f "$AUTOSTART_FILE" ]; then
    rm -f "$AUTOSTART_FILE"
    echo "Removed autostart entry"
fi

exit 0
