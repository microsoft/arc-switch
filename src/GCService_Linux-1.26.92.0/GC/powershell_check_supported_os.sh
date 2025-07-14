#!/bin/bash
#
# This script checks if the PowerShell 7.4.x / .NET 8 is supported by the OS
#

fail()
{
    ERRORMSG="$@"
    echo $ERRORMSG
    exit 1
}

get_linux_version() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        os_version=$VERSION_ID
    elif type lsb_release >/dev/null 2>&1; then
        os_version=$(lsb_release -sr)
    elif [ -f /etc/lsb-release ]; then
        . /etc/lsb-release
        os_version=$DISTRIB_RELEASE
    elif [ -f /etc/debian_version ]; then
        os_version=$(cat /etc/debian_version)
    else
        # Fall back to uname.
        os_version=$(uname -r)
    fi

    # If we fail to compute os version then return success.
    # If OS doesn't support .NET8, eventually PowerShell policy will fail.
    if [ -z $os_version ]; then
        echo "Unexpected error occurred while getting the distro version."
        exit 0
    fi
}


# To limit the ongoing maintanance cost of changing this script as and when we add new distros,
# We check only the known distros where GC is supported and .NET 8 is NOT supported.
# PowerShell requires .NET8.
# .NET 8 supported OS list can be found @ https://github.com/dotnet/core/blob/main/release-notes/8.0/supported-os.md

ubuntu_min_supported_os_version="16.04"
debian_min_supported_os_version="10"

get_linux_version
proc_version_output=$(cat /proc/version)
case $proc_version_output in
    *Debian*)
        min_supported_os_version=$debian_min_supported_os_version
        os_name="Debian"
        ;;
    *Ubuntu*)
        min_supported_os_version=$ubuntu_min_supported_os_version
        os_name="Ubuntu"
        ;;
    *)
        echo "PowerShell policies are supported on this OS version:$os_version"
        exit 0
        ;;
esac

if [ "$(printf '%s\n' "$os_version" "$min_supported_os_version" | sort -V | head -n1)" = "$min_supported_os_version" ]; then
    echo "PowerShell policies are supported on this OS:$os_name version:$os_version"
else
    fail "PowerShell policies are not supported on this OS:$os_name version:$os_version minimumSupportedVersion:$min_supported_os_version"
fi

exit 0
