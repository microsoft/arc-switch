#!/bin/bash
#
# This script is used to update Arc GC on Linux.
# Nothing needs to be migrated at the moment.
#
BASEDIR=$(dirname "$0")
echo $BASEDIR
find /var/lib/waagent/ -name gca.config -exec cp {} $BASEDIR  \; >/dev/null 2>&1

exit 0
