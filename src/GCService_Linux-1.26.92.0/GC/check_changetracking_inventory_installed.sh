#!/bin/bash

# Check if change tracking is enabled
for moffile in $(find /etc/opt/microsoft/omsagent -name change_tracking_inventory.mof); do
    # /etc/opt/microsoft/omsagent/sysconf/omsagent.d/change_tracking_inventory.mof is the default location.
    # ignore this path, change_tracking_inventory.mof is copied on this location by the omsagent installer
    # when change tracking is enabled it gets copied to /etc/opt/microsoft/omsagent/<workspaceid>/conf/omsagent.d/patch_management_inventory.mof location
    if [[ "$moffile" != *"/etc/opt/microsoft/omsagent/sysconf/omsagent.d/change_tracking_inventory.mof"* ]]; then
        echo "ChangeTracking is enabled : $moffile"
        exit 0
    fi
done
echo "ChangeTracking is not enabled."
exit 1