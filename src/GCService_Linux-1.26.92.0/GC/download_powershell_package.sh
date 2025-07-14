#!/bin/bash
# 
# This script downloads the PowerShell package on Linux.
# 

# GC path (Ex - /var/lib/waagent/Microsoft.GuestConfiguration.ConfigurationforLinux-1.26.4/GCAgent/GC) 
PS_HOME_PATH=$1
PS_DOWNLOAD_URL=$2
PS_DOWNLOAD_CHECKSUM=$3
PSDSC_MODULE_DOWNLOAD_URL=$4
PSDSC_MODULE_DOWNLOAD_CHECKSUM=$5
USAGE_ERROR='Missing argument. Usage: "<script> <GC_BIN_PATH> <PS_DOWNLOAD_URL> <PS_DOWNLOAD_CHECKSUM> <PSDSC_MODULE_DOWNLOAD_URL> <PSDSC_MODULE_DOWNLOAD_CHECKSUM>"'
if ( [ "x$PS_HOME_PATH" = "x" ] || [ "x$PS_DOWNLOAD_URL" == "x" ] || [ "x$PS_DOWNLOAD_CHECKSUM" = "x" ] || [ "x$PSDSC_MODULE_DOWNLOAD_URL" = "x" ] || [ "x$PSDSC_MODULE_DOWNLOAD_CHECKSUM" = "x" ] )
then
    echo $USAGE_ERROR
    exit 2
fi

PS_HOME_PATH=`realpath $PS_HOME_PATH 2> /dev/null`
# realpath is not found
if [ $? -eq 127 ]; then
    PS_HOME_PATH=$1
fi

if [ ! -d $PS_HOME_PATH ]; then
    echo "GC_BIN_PATH:$PS_HOME_PATH doesn't exist"
    exit 2
fi

MAX_DOWNLOAD_RETRY_COUNT=10
PACKAGE_NAME="Pwsh-7.2-preview-dsc-support.tar.xz"
PACKAGE_NAME_WITHOUT_EXT=`basename $PACKAGE_NAME .tar.xz`
EXPECTED_SHA256_CHECKSUM="2e58ab4999ab3607bce042f43809e88e189103e3165d7da3c65fdf787d9593a6"

if [ "x$PS_DOWNLOAD_URL" != "x" ]; then
    if [[ "$PS_DOWNLOAD_URL" == *tar.gz ]]; then
        PACKAGE_NAME=`basename $PS_DOWNLOAD_URL`
        PACKAGE_NAME_WITHOUT_EXT=`basename $PACKAGE_NAME .tar.gz`
        EXPECTED_SHA256_CHECKSUM=$PS_DOWNLOAD_CHECKSUM
    else
        echo "Invalid PS_DOWNLOAD_URL:$PS_DOWNLOAD_URL"
        exit 2
    fi
fi

# In priority order. Default is WCUS.
AVAILABLE_AZURE_STORAGE_REGIONS=('wcus'
    'we'
    'ase'
    'brs'
    'cid'
    'eus2'
    'ne'
    'scus'
    'uks'
    'wus2')

check_result() {
    if [ $1 -ne 0 ]; then
        echo $2
        rm -rf $PS_HOME_PATH/$PACKAGE_NAME
        exit $1
    fi
}

get_azure_storage_url() {
    CURRENT_REGION=$1
    AZURE_STORAGE_ENDPOINT="oaasguestconfig${CURRENT_REGION}s1"
    AZURE_STORAGE_URL="https://${AZURE_STORAGE_ENDPOINT}.blob.core.windows.net/powershellpkgs"
}

rotate_azure_storage_url() {
    RETRY_NUM=$1

    NUM_AVAILABLE_REGIONS=${#AVAILABLE_AZURE_STORAGE_REGIONS[@]}

    CURRENT_REGION_INDEX=$(( RETRY_NUM % NUM_AVAILABLE_REGIONS ))
    CURRENT_REGION=${AVAILABLE_AZURE_STORAGE_REGIONS[$CURRENT_REGION_INDEX]}
    get_azure_storage_url $CURRENT_REGION
}

download_package() {
    PACKAGE_NAME=$1
    PACKAGE_URL=$2
    DOWNLOAD_PACKAGE_NAME=$3
    DOWNLOAD_SUCCEEDED=false
    TOTAL_RETRY_SEC=0
    RETRY_SLEEP_INTERVAL_SEC=5
    RETRY_MAX_SEC=60
    CURL_MAX_SEC=120

    while [ $TOTAL_RETRY_SEC -lt $RETRY_MAX_SEC ] &&  [ "$DOWNLOAD_SUCCEEDED" = false ]; do
        if [ $TOTAL_RETRY_SEC -gt 0 ]; then
            echo "Retrying after $RETRY_SLEEP_INTERVAL_SEC seconds"
            sleep $RETRY_SLEEP_INTERVAL_SEC
        fi

        echo "Downloading package '$PACKAGE_NAME' from the URL '$PACKAGE_URL'"
        cd $PS_HOME_PATH

        # "curl" is present on all GC supported operating systems.
        # install_inspec.sh already takes dependency on "curl".
        if command -v curl &> /dev/null
        then
            echo "Downloading with curl"
            HTTP_RESPONSE_CODE=$(curl -L -sS -w "%{http_code}" -o $DOWNLOAD_PACKAGE_NAME -m $CURL_MAX_SEC $PACKAGE_URL)
            RESPONSE_CODE=$?
        elif command -v wget &> /dev/null # Use wget if curl is not available
        then
            echo "Downloading with wget"
            HTTP_RESPONSE_CODE=$(wget --server-response --tries=5 --timeout=$CURL_MAX_SEC $PACKAGE_URL  2>&1 | awk '/^  HTTP/{print $2}')
            RESPONSE_CODE=$?
        fi

        if [ $RESPONSE_CODE -eq 0 ]; then
            if [ $HTTP_RESPONSE_CODE -ne 200 ]; then
                echo "Download of package '$PACKAGE_NAME' failed with the HTTP response code '$HTTP_RESPONSE_CODE'"
            else
                DOWNLOAD_SUCCEEDED=true
            fi
        else
            echo "Download of package '$PACKAGE_NAME' failed with the response code '$RESPONSE_CODE'"
        fi

        TOTAL_RETRY_SEC+=$RETRY_SLEEP_INTERVAL_SEC
    done

    if [ "$DOWNLOAD_SUCCEEDED" = true ]; then
        echo "Download of package '$PACKAGE_NAME' succeeded"
        return 0
    else
        echo "Download of package '$PACKAGE_NAME' failed after retrying for $RETRY_MAX_SEC seconds"
        return 1
    fi
}

test_sha256_checksums_match() {
    FILE_TO_CHECK=$1
    EXPECTED_SHA256_CHECKSUM=$2

    echo "Comparing checksums with sha256sum"
    ACTUAL_SHA256_CHECKSUM=`sha256sum $FILE_TO_CHECK | awk '{ print $1 }'`
    test "x$ACTUAL_SHA256_CHECKSUM" = "x$EXPECTED_SHA256_CHECKSUM"
    CHECKSUM_RESULT=$?

    if [ $CHECKSUM_RESULT -ne 0 ]; then
        echo "PowerShell package checksum does not match. actual_checksum:$ACTUAL_SHA256_CHECKSUM expected_checksum:$EXPECTED_SHA256_CHECKSUM"
        return 1
    else
        echo "PowerShell package checksum matches expected checksum"
        return 0
    fi
}

download_and_validate_package() {
    PACKAGE_NAME=$1
    EXPECTED_SHA256_CHECKSUM=$2

    if [ "x$PS_DOWNLOAD_URL" != "x" ]; then
        DOWNLOAD_URL=$PS_DOWNLOAD_URL
    else
        DOWNLOAD_URL="$AZURE_STORAGE_URL/$PACKAGE_NAME"
    fi

    download_package $PACKAGE_NAME $DOWNLOAD_URL $PACKAGE_NAME
    if [ $? -ne 0 ]; then
        return 1
    else
        test_sha256_checksums_match "$PACKAGE_NAME" "$EXPECTED_SHA256_CHECKSUM"
        if [ $? -ne 0 ]; then
            echo "Removing downloaded $PACKAGE_NAME file since checksums do not match"
            rm -rf $PS_HOME_PATH/$PACKAGE_NAME
            return 2
        fi
    fi
    
    return 0
}

download_and_validate_package_with_retries() {
    PACKAGE_NAME=$1
    EXPECTED_SHA256_CHECKSUM=$2

    NUM_RETRIES=0
    DOWNLOAD_RESULT=1
 
    while [ $NUM_RETRIES -lt $MAX_DOWNLOAD_RETRY_COUNT ] && [ $DOWNLOAD_RESULT -ne 0 ]; do
        if [ "x$PS_DOWNLOAD_URL" = "x" ]; then
            rotate_azure_storage_url $NUM_RETRIES
        fi

        download_and_validate_package $PACKAGE_NAME $EXPECTED_SHA256_CHECKSUM
        DOWNLOAD_RESULT=$?
        ((NUM_RETRIES++))
    done

    check_result $DOWNLOAD_RESULT "Failed to download PowerShell"
}

download_ps_desiredstateconfiguration_module() {
    MODULE_URL=$1
    EXPECTED_SHA256_CHECKSUM=$2
    MODULE_FOLDER_NAME="Modules"
    PACKAGE_NAME="PSDesiredStateConfiguration.zip"
    cd "$PS_HOME_PATH/$MODULE_FOLDER_NAME"
    # Download PSDesiredStateConfiguration module
    wget $MODULE_URL -O $PACKAGE_NAME
    # validate the checksum
    test_sha256_checksums_match "$PACKAGE_NAME" "$EXPECTED_SHA256_CHECKSUM"
    if [ $? -ne 0 ]; then
        echo "Removing downloaded $PACKAGE_NAME file since checksums do not match"
        rm -rf $PS_HOME_PATH/$MODULE_FOLDER_NAME/$PACKAGE_NAME
        return 2
    fi

    mkdir -p PSDesiredStateConfiguration
    unzip ./$PACKAGE_NAME -d ./PSDesiredStateConfiguration

    # Remove unnecessary files
    module_files_to_remove=("PSDesiredStateConfiguration.zip" "PSDesiredStateConfiguration/_rels" \
                        "PSDesiredStateConfiguration/PSDesiredStateConfiguration.nuspec" "PSDesiredStateConfiguration/*.xml" \
                        "PSDesiredStateConfiguration/package")
    for i in ${!module_files_to_remove[@]};
    do
        module_file_to_remove=${module_files_to_remove[$i]}
        rm -rf $PS_HOME_PATH/$MODULE_FOLDER_NAME/$module_file_to_remove
    done
}

install_powershell() {
    if [ ! -f "$PS_HOME_PATH/System.Management.Automation.dll" ] ; then
        download_and_validate_package_with_retries $PACKAGE_NAME $EXPECTED_SHA256_CHECKSUM
        if [ "x$PS_DOWNLOAD_URL" = "x" ]; then
            tar xf $PACKAGE_NAME >/dev/null 2>&1
        else
            mkdir  "$PS_HOME_PATH/$PACKAGE_NAME_WITHOUT_EXT"
            tar xf $PACKAGE_NAME -C "$PS_HOME_PATH/$PACKAGE_NAME_WITHOUT_EXT" >/dev/null 2>&1
        fi

        if [ $? -ne 0 ]; then
            echo "Failed to uncompress $PACKAGE_NAME"
            rm -rf $PS_HOME_PATH/$PACKAGE_NAME
            rm -rf $PS_HOME_PATH/$PACKAGE_NAME_WITHOUT_EXT

            exit 2
        fi

        # Remove unnecessary files including System.Data.SqlClient.dll for CVE 2022-41064
        ps_files_to_remove=("libmi.so" "pwsh*" \
                            "ThirdPartyNotices.txt" "LICENSE.txt" \
                            "cs" "de" "es" "fr" "it" "ja" "ko" "pl" "pt-BR" "ru" "tr" "zh-Hans" "zh-Hant" \
                            "Modules/PowerShellGet" "Modules/PackageManagement" "Modules/PSReadLine" "ThirdPartyNotices.txt" "LICENSE.txt" \
                            "System.Data.SqlClient.dll")
        for i in ${!ps_files_to_remove[@]};
        do
            ps_file_to_remove=${ps_files_to_remove[$i]}
            rm -rf $PS_HOME_PATH/$PACKAGE_NAME_WITHOUT_EXT/$ps_file_to_remove
        done

        # copy the powershell payload
        cp -rf $PS_HOME_PATH/$PACKAGE_NAME_WITHOUT_EXT/* $PS_HOME_PATH/

        # Remove the downloaded payload, temp directory
        rm -rf $PS_HOME_PATH/$PACKAGE_NAME_WITHOUT_EXT*

        if [ ! -f "$PS_HOME_PATH/System.Management.Automation.dll" ]; then
            echo "Couldn't find SMA.dll in the PowerShell package"
            exit 2
        fi

        echo "PowerShell payload is downloaded successfully"

        download_ps_desiredstateconfiguration_module $PSDSC_MODULE_DOWNLOAD_URL $PSDSC_MODULE_DOWNLOAD_CHECKSUM
        echo "PSDesiredStateConfiguration module is downloaded successfully"
    else
        echo "PowerShell is already installed"
    fi
}

install_powershell
check_result $? "Failed to download PowerShell"

