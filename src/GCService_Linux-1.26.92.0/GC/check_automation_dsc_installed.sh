#!/bin/bash

# Check if automation dsc is installed and registered to pull server
if grep -Rq "agentsvc" /etc/opt/omi/conf/dsc/configuration/MetaConfig.mof
then
    echo "DSC is registered to pull server"
    exit 0
else
    echo "DSC is not registered to pull server"
fi
exit 1