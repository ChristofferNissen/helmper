#!/usr/bin/env bash

set -e

# Plugin installation directory
if [ "$HELM_PLUGIN_DIR" != "" ]; then
    PLUGIN_DIR="$HELM_PLUGIN_DIR"
else
    echo "Error: HELM_PLUGIN_DIR is not set"
    exit 1
fi

# Remove the binary
if [ -f "${PLUGIN_DIR}/bin/helmper" ]; then
    rm "${PLUGIN_DIR}/bin/helmper"
    echo "Removed plugin binary"
fi

# Remove the bin directory if empty
if [ -d "${PLUGIN_DIR}/bin" ] && [ "$(ls -A "$PLUGIN_DIR"/bin)" = "" ]; then
    rmdir "${PLUGIN_DIR}/bin"
    echo "Removed empty bin directory"
fi

echo "Plugin uninstallation completed successfully!"
