#!/bin/bash
# 
# This script installs Arc GC on Linux and starts the GC Arc service.
# 

GC_HOME_PATH="$PWD"
GC_EXE_PATH="$GC_HOME_PATH/gc_linux_service"
GC_SERVER_SOCKET_PATH="$GC_HOME_PATH/sockets"
SERVICE_TEMP_FOLDER_PATH="$GC_HOME_PATH/service_temp"


SERVICE_SCRIPTS_FOLDER_PATH="$GC_HOME_PATH/service_scripts"
SERVICE_CONTROLLER_PATH="$SERVICE_SCRIPTS_FOLDER_PATH/gca_service_controller"

GC_SERVICE_NAME="gcad"

GC_SYSTEMD_FILE_NAME="$GC_SERVICE_NAME.systemd"
GC_SYSTEMD_SOURCE_FILE_PATH="$SERVICE_SCRIPTS_FOLDER_PATH/$GC_SYSTEMD_FILE_NAME"
GC_SYSTEMD_TEMP_FILE_PATH="$SERVICE_TEMP_FOLDER_PATH/$GC_SYSTEMD_FILE_NAME"

GC_UPSTART_FILE_NAME="$GC_SERVICE_NAME.upstart"
GC_UPSTART_SOURCE_FILE_PATH="$SERVICE_SCRIPTS_FOLDER_PATH/$GC_UPSTART_FILE_NAME"
GC_UPSTART_TEMP_FILE_PATH="$SERVICE_TEMP_FOLDER_PATH/$GC_UPSTART_FILE_NAME"

GC_INITD_UPSTART_FILE_NAME="$GC_SERVICE_NAME.initd"
GC_INITD_UPSTART_SOURCE_FILE_INITD_PATH="$SERVICE_SCRIPTS_FOLDER_PATH/$GC_INITD_UPSTART_FILE_NAME"
GC_INITD_UPSTART_TEMP_FILE_INITD_PATH="$SERVICE_TEMP_FOLDER_PATH/$GC_INITD_UPSTART_FILE_NAME"


POWERSHELL_CONFIG_PATH="$GC_HOME_PATH/powershell.config.json"

SYSTEMD_UNIT_DIR=""
SYSTEM_SERVICE_CONTROLLER=""

LINUX_DISTRO=""

print_error() {
  echo "[$(date +'%Y-%m-%dT%H:%M:%S%z')]: $@" >&2
}

check_result() {
    if [ $1 -ne 0 ]; then
        print_error $2
        exit $1
    fi
}

compareversion () {
    if [[ $1 == $2 ]]
    then
        return 0
    fi

    # Sanitize version strings - extract only numbers and dots
    local clean_version1=$(echo "$1" | sed 's/[^0-9.].*$//')
    local clean_version2=$(echo "$2" | sed 's/[^0-9.].*$//')

    local IFS=.
    local i version1=($clean_version1) version2=($clean_version2)

    # Fill zeros in version1 if its length is less than version2
    for ((i=${#version1[@]}; i<${#version2[@]}; i++))
    do
        version1[i]=0
    done

    for ((i=0; i<${#version1[@]}; i++))
    do
        if [[ -z ${version2[i]} ]]
        then
            # Fill zeros in version2 if its length is less than version1
            version2[i]=0
        fi

        # compare the version digits
        if ((10#${version1[i]} > 10#${version2[i]}))
        then
            return 1
        fi
        if ((10#${version1[i]} < 10#${version2[i]}))
        then
            return 2
        fi
    done
    return 0
}

get_linux_version() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        LINUX_DISTRO_VERSION=$VERSION_ID
    elif type lsb_release >/dev/null 2>&1; then
        LINUX_DISTRO_VERSION=$(lsb_release -sr)
    elif [ -f /etc/lsb-release ]; then
        . /etc/lsb-release
        LINUX_DISTRO_VERSION=$DISTRIB_RELEASE
    elif [ -f /etc/debian_version ]; then
        LINUX_DISTRO_VERSION=$(cat /etc/debian_version)
    else
        # Fall back to uname.
        LINUX_DISTRO_VERSION=$(uname -r)
    fi

    if [ -z $LINUX_DISTRO_VERSION ]; then
        print_error "Unexpected error occurred while getting the distro version."
        exit 1
    fi
    echo "Linux distribution version is $LINUX_DISTRO_VERSION."
}

check_linux_distro() {
    if [ ! -z $LINUX_DISTRO ]; then
        return
    fi

    get_linux_version

    VERSION_OUTPUT=$(cat /proc/version)
    ARCHITECTURE_OUTPUT=$(uname -m)
    
    if [[ $ARCHITECTURE_OUTPUT = "aarch64" ]]; then
        if [[ $VERSION_OUTPUT = *"Ubuntu"* ]]; then
            LINUX_DISTRO="Ubuntu"
            MIN_SUPPORTED_DISTRO_VERSION="16.04" 
        elif [[ $VERSION_OUTPUT = *"Red Hat"* ]]; then
            LINUX_DISTRO="Red Hat"
            MIN_SUPPORTED_DISTRO_VERSION="8.0"
        elif [[ $VERSION_OUTPUT = *"SUSE"* ]]; then
            LINUX_DISTRO="SUSE"
            MIN_SUPPORTED_DISTRO_VERSION="15.0"
        elif [[ $VERSION_OUTPUT = *"CentOS"* ]]; then
            LINUX_DISTRO="CentOS"
            MIN_SUPPORTED_DISTRO_VERSION="8.0"
        elif [[ $VERSION_OUTPUT = *"Debian"* ]]; then
            LINUX_DISTRO="Debian"
            MIN_SUPPORTED_DISTRO_VERSION="9.0"
        elif [[ $VERSION_OUTPUT = *"Mariner"* ]]; then
            LINUX_DISTRO="Mariner"
            MIN_SUPPORTED_DISTRO_VERSION="2.0"
        elif [[ $VERSION_OUTPUT = *"nxos"* ]]; then
            LINUX_DISTRO="NXOS"
            MIN_SUPPORTED_DISTRO_VERSION="10.0"
        else
            print_error "Unexpected Linux distribution. Expected Linux distributions include only Ubuntu, Red Hat, SUSE, CentOS, and Debian."
            # Exit with error code 51 (The extension is not supported on this OS)
            exit 51
        fi
    else

        if [[ $VERSION_OUTPUT = *"Ubuntu"* ]]; then
            LINUX_DISTRO="Ubuntu"
            MIN_SUPPORTED_DISTRO_VERSION="14.04"
        elif [[ $VERSION_OUTPUT = *"Red Hat"* ]]; then
            LINUX_DISTRO="Red Hat"
            MIN_SUPPORTED_DISTRO_VERSION="7.0"
        elif [[ $VERSION_OUTPUT = *"SUSE"* ]]; then
            LINUX_DISTRO="SUSE"
            MIN_SUPPORTED_DISTRO_VERSION="12.0"
        elif [[ $VERSION_OUTPUT = *"CentOS"* ]]; then
            LINUX_DISTRO="CentOS"
            MIN_SUPPORTED_DISTRO_VERSION="7.0"
        elif [[ $VERSION_OUTPUT = *"Debian"* ]]; then
            LINUX_DISTRO="Debian"
            MIN_SUPPORTED_DISTRO_VERSION="8.0"
        elif [[ $VERSION_OUTPUT = *"Mariner"* ]]; then
            LINUX_DISTRO="Mariner"
            MIN_SUPPORTED_DISTRO_VERSION="1.0"
        elif [[ $VERSION_OUTPUT = *"nxos"* ]]; then
            LINUX_DISTRO="NXOS"
            MIN_SUPPORTED_DISTRO_VERSION="10.0"
        else
            print_error "Unexpected Linux distribution. Expected Linux distributions include only Ubuntu, Red Hat, SUSE, CentOS, and Debian."
            # Exit with error code 51 (The extension is not supported on this OS)
            exit 51
        fi
    fi

    compareversion $LINUX_DISTRO_VERSION $MIN_SUPPORTED_DISTRO_VERSION
    if [[ $? -eq 2 ]]; then
        print_error "Unsupported $LINUX_DISTRO version $LINUX_DISTRO_VERSION. $LINUX_DISTRO version should be greater or equal than $MIN_SUPPORTED_DISTRO_VERSION."
        # Exit with error code 51 (The extension is not supported on this OS)
        exit 51
    fi

    echo "Linux distribution is $LINUX_DISTRO."
}

get_system_service_controller() {
    if [ ! -z $SYSTEM_SERVICE_CONTROLLER ]; then
        return
    fi

    COMM_OUTPUT=$(cat /proc/1/comm)

    if [[ $COMM_OUTPUT = *"systemd"* ]]; then
        SYSTEM_SERVICE_CONTROLLER="systemd"
    elif [[ $COMM_OUTPUT = *"init"* ]]; then
        SYSTEM_SERVICE_CONTROLLER="upstart"
    else
        print_error "Unexpected system service controller. Expected system service controllers are systemd and upstart."
        exit 1
    fi

    echo "Service controller is $SYSTEM_SERVICE_CONTROLLER."
}

resolve_systemd_paths() {
    local UNIT_DIR_LIST="/usr/lib/systemd/system /lib/systemd/system"

    # Be sure systemctl lives where we expect it to
    if [ ! -f /bin/systemctl ]; then
        print_error "FATAL: Unable to locate systemctl program"
        exit 1
    fi

    # Find systemd unit directory
    for i in ${UNIT_DIR_LIST}; do
        if [ -d $i ]; then
            SYSTEMD_UNIT_DIR=${i}
            return 0
        fi
    done

    # Didn't find unit directory, that's fatal
    print_error "FATAL: Unable to resolve systemd unit directory"
    exit 1
}

create_systemd_config_file() {
    # Remove any old temp systemd configuration file that may exist
    if [ -f $GC_SYSTEMD_TEMP_FILE_PATH ]; then
        rm -f $GC_SYSTEMD_TEMP_FILE_PATH
    fi

    if [ ! -d $SERVICE_TEMP_FOLDER_PATH ]; then
        mkdir $SERVICE_TEMP_FOLDER_PATH
    fi

    # Replace the pid file and exe file paths in the systemd configuration file
    cat $GC_SYSTEMD_SOURCE_FILE_PATH | sed "s@<EXE_FILE_PATH>@$GC_EXE_PATH@g" > $GC_SYSTEMD_TEMP_FILE_PATH;

    # Set the new temp systemd configuration file to the correct permissions  
    chmod 644 $GC_SYSTEMD_TEMP_FILE_PATH;
}

install_systemd_service() {
    echo "Found systemd service controller...for Arc GC Service"
    resolve_systemd_paths
    create_systemd_config_file
    cp -f $GC_SYSTEMD_TEMP_FILE_PATH ${SYSTEMD_UNIT_DIR}/gcad.service
    chmod 644 ${SYSTEMD_UNIT_DIR}/gcad.service
    /bin/systemctl daemon-reload
    /bin/systemctl enable gcad 2>&1
    echo "Service configured through systemd service controller. Gc Service"
}

create_upstart_config_file() {

    INIT_SYSTEM=$(ps -p 1 -o comm=)
    # if systemV use GC_INITD_UPSTART_SOURCE_FILE_INITD_PATH
    if [ "$INIT_SYSTEM" = "init" ] || [ "$INIT_SYSTEM" = "sysvinit" ]; then
        GC_UPSTART_SOURCE_FILE_PATH="$GC_INITD_UPSTART_SOURCE_FILE_INITD_PATH"
        GC_UPSTART_TEMP_FILE_PATH="$GC_INITD_UPSTART_TEMP_FILE_INITD_PATH"
    fi

    # Remove any old temp upstart configuration file that may exist
    if [ -f $GC_UPSTART_TEMP_FILE_PATH ]; then
        rm -f $GC_UPSTART_TEMP_FILE_PATH
    fi

    if [ ! -d $SERVICE_TEMP_FOLDER_PATH ]; then
        mkdir $SERVICE_TEMP_FOLDER_PATH
    fi

    # Replace the exe file path in the upstart configuration file
    cat $GC_UPSTART_SOURCE_FILE_PATH | sed "s@<GC_EXE_PATH>@$GC_EXE_PATH@g" > $GC_UPSTART_TEMP_FILE_PATH;

    # Set the new temp upstart configuration file to the correct permissions  
    chmod 644 $GC_UPSTART_TEMP_FILE_PATH;
}

install_upstart_service() {
    INIT_SYSTEM=$(ps -p 1 -o comm=)
    if [ -x /sbin/initctl -a -f /etc/init/networking.conf ]; then
        # If we have /sbin/initctl, we have upstart.
        # Note that the upstart script requires networking,
        # so only use upstart if networking is controlled by upstart (not the case in RedHat 6)
        echo "Found upstart service controller with networking..."
        create_upstart_config_file
        cp -f $GC_UPSTART_TEMP_FILE_PATH /etc/init/gcad.conf
        chmod 644 /etc/init/gcad.conf
        
        # initctl registers it with upstart
        initctl reload-configuration
        echo "Service configured through upstart service controller."
    elif [ "$INIT_SYSTEM" = "init" ] || [ "$INIT_SYSTEM" = "sysvinit" ]; then
        # If we have sysvinit, we can use the init.d script.
        echo "Found sysvinit service controller..."
        create_upstart_config_file
        cp -f $GC_UPSTART_TEMP_FILE_PATH /etc/init.d/gcad
        chmod 755 /etc/init.d/gcad
        
        # Register the service with sysvinit
        update-rc.d gcad defaults
        echo "Service configured through sysvinit service controller."
    else
        print_error "Upstart service controller does not have control of the networking service."
        exit 1
    fi
}

install_gc_service() {

    # Set the GC service controller to be executable
    chown root $SERVICE_CONTROLLER_PATH
    chmod 700 $SERVICE_CONTROLLER_PATH

    $SERVICE_CONTROLLER_PATH stop
    echo "Configuring Arc GC service ..."
    
    pidof systemd 1> /dev/null 2> /dev/null
    if [ $? -eq 0 ]; then
        install_systemd_service
    else
        get_system_service_controller
        case "$SYSTEM_SERVICE_CONTROLLER" in
        "systemd")
            install_systemd_service
            ;;
        "upstart")
            install_upstart_service $1
            ;;
        *) echo "Unrecognized system service controller to configure Arc GC service."
            exit 1
            ;;
        esac
    fi
}

install_gc() {
    chown root $GC_EXE_PATH
    check_result $? "Setting owner of gca_linux_service file failed"

    chmod 700 $GC_EXE_PATH
    check_result $? "Setting permissions of gca_linux_service file failed"

    chown root $GC_HOME_PATH/*.sh
    check_result $? "Setting owner of .sh files failed"

    chmod 700 $GC_HOME_PATH/*.sh
    check_result $? "Setting permissions of .sh files failed"

    cat <<EOF > "$GC_HOME_PATH/gc.config"
    {"ServiceType" : "GCArc"}
EOF

    mkdir -p $GC_SERVER_SOCKET_PATH
    check_result $? "Creating sockets directory failed"
    chmod 700 $GC_SERVER_SOCKET_PATH
    check_result $? "Setting permissions of sockets directory failed"
    chown root $GC_SERVER_SOCKET_PATH
    check_result $? "Changing ownership of sockets directory failed"

    install_gc_service
}


create_guest_config_folder() {
    mkdir -p "/var/lib/GuestConfig"
    chmod 700 "/var/lib/GuestConfig"
}

create_powershell_config_file() {
    # disable powershell verbose debug logging, these logs unnecessarily appears in syslog files
    echo "{\"LogLevel\":\"Critical\"}" > $POWERSHELL_CONFIG_PATH
}

create_guest_config_folder
check_result $? "Failed to create guest config data directory"
install_gc
check_result $? "Installation of Arc GC service failed"
create_powershell_config_file