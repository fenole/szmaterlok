#!/bin/bash

BIN_NAME="szmaterlok"
BIN_PATH="./cmd/szmaterlok"
WATCH_SLEEP=3

function last_update {
    echo $(stat -t '%s' -f '%Sm' $1)
}

function go:watch {
    go:build
    ./$BIN_NAME &
    binpid=$!
    bin_last_updated=$(last_update ./$BIN_NAME)

    while true; do
        # read all recursively files from current directory with
        # extensions that matter
        files=( $(find . -name '*.go' \
            -or -name '*.mod' \
            -or -name '*.sum' \
            -or -name '*.json' \
            -or -name '*.js' \
            -or -name '*.html' \
            -or -name '*.css') )

        # establish maximum last update time
        max=$(last_update ${files[0]})
        for t in "${files[@]}"; do
            last_updated=$(last_update $t)

            (( last_updated > max )) && max=$last_updated
        done

        # if maximum is greater than update time of binary
        # rerun szmaterlok executable
        if (( $max > $bin_last_updated )); then
            kill -INT $binpid
            go:build
            ./$BIN_NAME &
            binpid=$!
            bin_last_updated=$(last_update ./$BIN_NAME)
        fi

        # wait for next iteration
        sleep $WATCH_SLEEP
    done
}

function go:build {
    go build -o $BIN_NAME $BIN_PATH
}

function go:run {
    go:build
    ./$BIN_NAME
}

function go:test {
    go test -v ./...
}

function default {
    go:build
}

function help {
    echo "$0 <task> <args>"
    echo "Tasks:"
    compgen -A function | cat -n
}

TIMEFORMAT="Task completed in %3lR"
time ${@:-default}
