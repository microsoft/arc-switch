#!/bin/bash
# 
# This script enables Arc GC on Linux by starting the gcad service.
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

start_gc_service() {
    $SERVICE_CONTROLLER_PATH start
    check_result $? "Starting the Arc GC service failed"
}

enable_gc() {
    start_gc_service
}

enable_gc
