
# When sourced, creates an Alfred-like environment needed by modd
# and ./bin/build (which sources the file itself)

# getvar <name> | Read a value from info.plist
getvar() {
    local v="$1"
    /usr/libexec/PlistBuddy -c "Print :$v" info.plist
}

export alfred_workflow_bundleid=$( getvar "bundleid" )
export alfred_workflow_version=$( getvar "version" )
export alfred_workflow_name=$( getvar "name" )

export alfred_workflow_cache="${HOME}/Library/Caches/com.runningwithcrayons.Alfred/Workflow Data/${alfred_workflow_bundleid}"
export alfred_workflow_data="${HOME}/Library/Application Support/Alfred/Workflow Data/${alfred_workflow_bundleid}"

if [[ ! -f "$HOME/Library/Preferences/com.runningwithcrayons.Alfred.plist" ]]; then
    export alfred_workflow_cache="${HOME}/Library/Caches/com.runningwithcrayons.Alfred-3/Workflow Data/${alfred_workflow_bundleid}"
    export alfred_workflow_data="${HOME}/Library/Application Support/Alfred 3/Workflow Data/${alfred_workflow_bundleid}"
    export alfred_version="3.8.1"
fi

export SCHEDULE_DAYS=$( getvar "variables:SCHEDULE_DAYS" )
export EVENT_CACHE_MINS=$( getvar "variables:EVENT_CACHE_MINS" )
