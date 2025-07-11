#!/bin/bash
# 
# This script uninstalls Extension Management on Linux.
# 

GC_HOME_PATH="$PWD"
SERVICE_TEMP_FOLDER_PATH="$GC_HOME_PATH/service_temp"

SERVICE_SCRIPTS_FOLDER_PATH="$GC_HOME_PATH/service_scripts"
SERVICE_CONTROLLER_PATH_EXT="$SERVICE_SCRIPTS_FOLDER_PATH/ext_service_controller"

EXT_SERVICE_NAME="extd"

SYSTEMD_UNIT_DIR=""
SYSTEM_SERVICE_CONTROLLER=""

print_error() {
  echo "[$(date +'%Y-%m-%dT%H:%M:%S%z')]: $@" >&2
}

check_result() {
    if [ $1 -ne 0 ]; then
        print_error $2
        exit $1
    fi
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

uninstall_systemd_service() {
    SERVICE=$1
    resolve_systemd_paths
    if [ -f ${SYSTEMD_UNIT_DIR}/${SERVICE}.service ]; then
        echo "Unconfiguring ${SERVICE} (systemd) service ..."
        /bin/systemctl disable ${SERVICE}
        rm -f ${SYSTEMD_UNIT_DIR}/${SERVICE}.service
        /bin/systemctl daemon-reload
    fi
}

uninstall_upstart_service() {
    SERVICE=$1
    echo "Unconfiguring ${SERVICE} (upstart) service ..."
    rm -f /usr/init/${SERVICE}.conf
    initctl reload-configuration
}

uninstall_init.daemon_service() {
    SERVICE=$1
    echo "Unconfiguring ${SERVICE} (init.d) service ..."
    if [ -f /etc/init.d/${SERVICE} ]; then
        # remove /etc/rc.d links for the service
        for i in /etc/rc*.d/S??${SERVICE} /etc/rc*.d/K??${SERVICE}; do
            if [ -L "$i" ] || [ -f "$i" ]; then
            rm -f "$i"
            fi
        done
        # remove the init.d script
        update-rc.d -f ${SERVICE} remove
        rm -f /etc/init.d/${SERVICE}
    fi
}

remove_service() {
    SERVICE=$1
    SERVICE_CONTROLLER_PATH_LOCAL=$2
    if [ -z "$SERVICE" ]; then
        echo "FATAL: remove_service requires parameter (service name)" 1>&2
        exit 1
    fi

    # Stop the service in case it's running
    $SERVICE_CONTROLLER_PATH_LOCAL stop

    # Registered as a systemd service?
    #
    # Note: We've never deployed systemd unit files automatically in the %Files
    # section. Thus, for systemd services, it's safe to remove the file.
    
    if pidof systemd 1> /dev/null 2> /dev/null; then
        uninstall_systemd_service $SERVICE
    elif [ -f /etc/init/${SERVICE}.conf ]; then
        uninstall_upstart_service $SERVICE
    else
        get_system_service_controller
        case "$SYSTEM_SERVICE_CONTROLLER" in
        "systemd")
            uninstall_systemd_service $SERVICE
            ;;
        "upstart")
            if [ -f /etc/init/${SERVICE}.conf ]; then
                uninstall_upstart_service $SERVICE
            elif [ -f /etc/init.d/${SERVICE} ]; then
                uninstall_init.daemon_service $SERVICE
            fi
            ;;
        *) echo "Unrecognized system service controller to unregister ${SERVICE} service."
            exit 1
            ;;
        esac
    fi

    return 0
}

remove_gc_service() {
    remove_service $EXT_SERVICE_NAME $SERVICE_CONTROLLER_PATH_EXT
    [ -f /etc/init.d/$EXT_SERVICE_NAME ] && rm /etc/init.d/$EXT_SERVICE_NAME
    [ -f /etc/init/$EXT_SERVICE_NAME.conf ] && rm /etc/init/$EXT_SERVICE_NAME.conf
    return 0
}

remove_gc_service