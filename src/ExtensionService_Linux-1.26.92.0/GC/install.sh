#!/bin/bash
# 
# This script installs Extension Management on Linux and starts the extd service.
# 
           
GC_HOME_PATH="$PWD" # /opt/GC_Ext/GC
GC_EXE_PATH="$GC_HOME_PATH/gc_linux_service" # /opt/GC_Ext/GC/gc_linux_service
EXT_SERVER_SOCKET_PATH="$GC_HOME_PATH/sockets" # /opt/GC_Ext/GC/sockets
SERVICE_TEMP_FOLDER_PATH="$GC_HOME_PATH/service_temp" # /opt/GC_Ext/GC/service_temp


SERVICE_SCRIPTS_FOLDER_PATH="$GC_HOME_PATH/service_scripts" # /opt/GC_Ext/GC/service_scripts
SERVICE_CONTROLLER_PATH_EXT="$SERVICE_SCRIPTS_FOLDER_PATH/ext_service_controller" # /opt/GC_Ext/GC/service_scripts/ext_service_controller

EXT_SERVICE_NAME="extd"

EXT_SYSTEMD_FILE_NAME="$EXT_SERVICE_NAME.systemd" # extd.service
EXT_SYSTEMD_SOURCE_FILE_PATH="$SERVICE_SCRIPTS_FOLDER_PATH/$EXT_SYSTEMD_FILE_NAME" # /opt/GC_Ext/GC/service_scripts/extd.service
EXT_SYSTEMD_TEMP_FILE_PATH="$SERVICE_TEMP_FOLDER_PATH/$EXT_SYSTEMD_FILE_NAME" # /opt/GC_Ext/GC/service_temp/extd.service

EXT_UPSTART_FILE_NAME="$EXT_SERVICE_NAME.upstart"
EXT_UPSTART_SOURCE_FILE_PATH="$SERVICE_SCRIPTS_FOLDER_PATH/$EXT_UPSTART_FILE_NAME" # /opt/GC_Ext/GC/service_scripts/extd.upstart
EXT_UPSTART_SOURCE_FILE_INITD_PATH="$SERVICE_SCRIPTS_FOLDER_PATH/extd_initd.upstart" # /opt/GC_Ext/GC/service_scripts/extd_initd.upstart
EXT_UPSTART_TEMP_FILE_PATH="$SERVICE_TEMP_FOLDER_PATH/$EXT_UPSTART_FILE_NAME" # /opt/GC_Ext/GC/service_temp/extd.upstart
EXT_UPSTART_TEMP_FILE_INITD_PATH="$SERVICE_TEMP_FOLDER_PATH/extd_initd.upstart" # /opt/GC_Ext/GC/service_temp/extd_initd.upstart

SYSTEMD_UNIT_DIR=""
SYSTEM_SERVICE_CONTROLLER=""

LINUX_DISTRO=""

print_error() {
  echo "[$(date +'%Y-%m-%dT%H:%M:%S%z')]: $@" >&2
}

check_result() {
    echo "check_result() - Entered"
    if [ $1 -ne 0 ]; then
        print_error $2
        exit $1
    fi
    echo "check_result() - Exit"
}

get_system_service_controller() {
    if [ ! -z $SYSTEM_SERVICE_CONTROLLER ]; then
        return
    fi

    COMM_OUTPUT=$(cat /proc/1/comm) # on Cisco Linux it returns "init"

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
    echo "resolve_systemd_paths() - Entered"
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
    echo "resolve_systemd_paths() - Exit 1"
    exit 1
}

create_systemd_config_file() {
    # Remove any old temp systemd configuration file that may exist
    if [ -f $EXT_SYSTEMD_TEMP_FILE_PATH ]; then
        rm -f $EXT_SYSTEMD_TEMP_FILE_PATH
    fi

    if [ ! -d $SERVICE_TEMP_FOLDER_PATH ]; then
        mkdir $SERVICE_TEMP_FOLDER_PATH
    fi

    # Replace the pid file and exe file paths in the systemd configuration file
    cat $EXT_SYSTEMD_SOURCE_FILE_PATH | sed "s@<EXE_FILE_PATH>@$GC_EXE_PATH@g" > $EXT_SYSTEMD_TEMP_FILE_PATH;

    # Set the new temp systemd configuration file to the correct permissions  
    chmod 644 $EXT_SYSTEMD_TEMP_FILE_PATH;
}

install_systemd_service() {
    echo "Found systemd service controller...for Extension Service"
    resolve_systemd_paths
    create_systemd_config_file
    cp -f $EXT_SYSTEMD_TEMP_FILE_PATH ${SYSTEMD_UNIT_DIR}/extd.service
    chmod 644 ${SYSTEMD_UNIT_DIR}/extd.service
    /bin/systemctl daemon-reload
    /bin/systemctl enable extd 2>&1
    echo "Service configured through systemd service controller. Extension Service"
}

create_upstart_config_file() {
    INIT_SYSTEM=$(ps -p 1 -o comm=)
    echo "create_upstart_config_file() - Entered"

    # if systemV use EXT_UPSTART_SOURCE_FILE_INITD_PATH
    if [ "$INIT_SYSTEM" = "init" ] || [ "$INIT_SYSTEM" = "sysvinit" ]; then
        EXT_UPSTART_SOURCE_FILE_PATH="$EXT_UPSTART_SOURCE_FILE_INITD_PATH"
        EXT_UPSTART_TEMP_FILE_PATH="$EXT_UPSTART_TEMP_FILE_INITD_PATH"
    fi

    # Remove any old temp upstart configuration file that may exist
    if [ -f $EXT_UPSTART_TEMP_FILE_PATH ]; then # /opt/GC_Ext/GC/service_temp/extd.upstart
        rm -f $EXT_UPSTART_TEMP_FILE_PATH
    fi

    if [ ! -d $SERVICE_TEMP_FOLDER_PATH ]; then # /opt/GC_Ext/GC/service_temp
        mkdir $SERVICE_TEMP_FOLDER_PATH
    fi

    # Replace the exe file path in the upstart configuration file
    #  /opt/GC_Ext/GC/service_scripts/extd.upstart    /opt/GC_Ext/GC/gc_linux_service   /opt/GC_Ext/GC/service_temp/extd.upstart



    echo "create_upstart_config_file() - Replacing exe file path in upstart configuration file"
    echo "create_upstart_config_file() - Source file path: EXT_UPSTART_SOURCE_FILE_PATH=$EXT_UPSTART_SOURCE_FILE_PATH"
    echo "create_upstart_config_file() - Temp file path: EXT_UPSTART_TEMP_FILE_PATH=$EXT_UPSTART_TEMP_FILE_PATH"
    echo "create_upstart_config_file() - GC_EXE_PATH=$GC_EXE_PATH"
    cat $EXT_UPSTART_SOURCE_FILE_PATH | sed "s@<GC_EXE_PATH>@$GC_EXE_PATH@g" > $EXT_UPSTART_TEMP_FILE_PATH;

    # Set the new temp upstart configuration file to the correct permissions
    echo "create_upstart_config_file() - Setting permissions for temp upstart configuration file: $EXT_UPSTART_TEMP_FILE_PATH"
    chmod 644 $EXT_UPSTART_TEMP_FILE_PATH;
    echo "create_upstart_config_file() - Exit"
}

install_upstart_service() {
    echo "install_upstart_service() - Entered"
    INIT_SYSTEM=$(ps -p 1 -o comm=)
    echo "install_upstart_service() - INIT_SYSTEM is $INIT_SYSTEM"
    if [ -x /sbin/initctl -a -f /etc/init/networking.conf ]; then
        # If we have /sbin/initctl, we have upstart.
        # Note that the upstart script requires networking,
        # so only use upstart if networking is controlled by upstart (not the case in RedHat 6)
        echo "Found upstart service controller with networking..."
        create_upstart_config_file
        #      /opt/GC_Ext/GC/service_temp/extd.upstart
        #      /opt/GC_Ext/GC/service_scripts/extd_initd.upstart
        cp -f $EXT_UPSTART_TEMP_FILE_PATH /etc/init.d/extd.conf
        chmod 644 /etc/init/extd.conf
        
        # initctl registers it with upstart
        initctl reload-configuration
        echo "Service configured through upstart service controller."
    elif [ "$INIT_SYSTEM" = "init" ] || [ "$INIT_SYSTEM" = "sysvinit" ]; then
        echo "install_upstart_service() - Found sysvinit service controller..."
        create_upstart_config_file
        #      /opt/GC_Ext/GC/service_temp/extd.upstart
        echo "install_upstart_service() - Copy $SERVICE_SCRIPTS_FOLDER_PATH/extd_initd.upstart to /etc/init.d/extd.conf"
        cp -f "$SERVICE_SCRIPTS_FOLDER_PATH/extd_initd.upstart" /etc/init.d/extd.conf
        echo "install_upstart_service() - Setting up init.d service for extd"
        
        chmod 755 /etc/init.d/extd.conf
        # create symlink to /etc/rc3.d and rc5.d
        echo "Creating symlinks for init.d service in /etc/rc3.d and /etc/rc5.d"
        ln -sf /etc/init.d/extd.conf /etc/rc3.d/S99extd
        ln -sf /etc/init.d/extd.conf /etc/rc5.d/S99extd
        # create symlink to  rc0.d, rc1.d, rc2.d, rc6.d
        echo "Creating symlinks for init.d service in /etc/rc0.d, /etc/rc1.d, /etc/rc2.d, and /etc/rc6.d"
        ln -sf /etc/init.d/extd.conf /etc/rc0.d/K01extd
        ln -sf /etc/init.d/extd.conf /etc/rc1.d/K01extd
        ln -sf /etc/init.d/extd.conf /etc/rc2.d/K01extd
        ln -sf /etc/init.d/extd.conf /etc/rc6.d/K01extd
    else
        print_error "Upstart service controller does not have control of the networking service."
        exit 1
    fi
    echo "install_upstart_service() - Exit"
}

install_gc_service() { 
    echo "install_gc_service() - Entered"

    # Set the EXT service controller to be executable
    #           /opt/GC_Ext/GC/service_scripts/ext_service_controller
    chown root $SERVICE_CONTROLLER_PATH_EXT
    chmod 700 $SERVICE_CONTROLLER_PATH_EXT

    $SERVICE_CONTROLLER_PATH_EXT stop
    echo "Configuring EXT service ..."
    
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
            echo "install_gc_service() - Using upstart service controller"
            install_upstart_service
            ;;
        *) echo "Unrecognized system service controller to configure EXTD service."
            exit 1
            ;;
        esac
    fi
    echo "install_gc_service() - Exit"
}

install_gc() {
    echo "install_gc() - Entered"
    chown root $GC_EXE_PATH
    check_result $? "Setting owner of gc_linux_service file failed"

    chmod 700 $GC_EXE_PATH
    check_result $? "Setting permissions of gc_linux_service file failed"

    chown root $GC_HOME_PATH/*.sh
    check_result $? "Setting owner of .sh files failed"

    chmod 700 $GC_HOME_PATH/*.sh
    check_result $? "Setting permissions of .sh files failed"

    cat <<EOF > "$GC_HOME_PATH/gc.config"
    {"ServiceType" : "Extension"}
EOF
    mkdir -p $EXT_SERVER_SOCKET_PATH
    check_result $? "Creating extension sockets directory failed"
    chmod 700 $EXT_SERVER_SOCKET_PATH
    check_result $? "Setting permissions of extension sockets directory failed"
    chown root $EXT_SERVER_SOCKET_PATH
    check_result $? "Changing ownership of extension sockets directory failed"

    install_gc_service
    echo "install_gc() - Exit"
}

create_guest_config_folder() {
    echo "create_guest_config_folder() - Entered"
    mkdir -p "/var/lib/GuestConfig"
    chmod 700 "/var/lib/GuestConfig"
    echo "Created guest config data directory at /var/lib/GuestConfig"

    echo "create_guest_config_folder() - Exit"
}

create_guest_config_folder
check_result $? "Failed to create guest config data directory"
install_gc
check_result $? "Installation of EXTD service failed"
