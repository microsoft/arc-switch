#!/bin/bash
# 
# This script disables Arc GC on Linux by stopping the gcad service.
# 

GC_HOME_PATH="$PWD"
SERVICE_SCRIPTS_FOLDER_PATH="$GC_HOME_PATH/service_scripts"
SERVICE_CONTROLLER_PATH="$SERVICE_SCRIPTS_FOLDER_PATH/gca_service_controller"

print_error() {
  echo "[$(date +'%Y-%m-%dT%H:%M:%S%z')]: $@" >&2
}

check_result() {
    if [ $1 -ne 0 ]; then
        print_error $2
        exit $1
    fi
}

stop_gc_service() {
    $SERVICE_CONTROLLER_PATH stop
    check_result $? "Stopping the Arc GC service failed"
}

disable_gc() {
    stop_gc_service
}

disable_gc
