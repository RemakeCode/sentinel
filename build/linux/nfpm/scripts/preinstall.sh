#!/bin/bash

# Stop any running instances of sentinel before installation/update.
# This ensures that files are not busy and the new version can be installed cleanly.
if pgrep -x "sentinel" >/dev/null 2>&1; then
  echo "Found running sentinel processes. Stopping them..."
  # Try to terminate gracefully first, then kill if needed after a short delay
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

exit 0
