#!/bin/zsh

BIN_NAME="szmaterlok"
BIN_PATH="./cmd/szmaterlok"
WATCH_SLEEP=3

function last_update {
    echo $(stat -t '%s' -f '%Sm' $1)
}

function go:watch { # watch for changes and rebuild go binaries
    go:build
    ./$BIN_NAME &
    binpid=$!
    bin_last_updated=$(last_update ./$BIN_NAME)

    max=0
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
        for t in $files; do
            last_updated=$(last_update $t)

            if (( $last_updated > $max )); then 
                max=$last_updated
            fi
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

function go:build { # build go binaries
    go build -o $BIN_NAME $BIN_PATH
}

function go:run { # run go backend server
    go:build
    ./$BIN_NAME
}

function go:test { # run go unit tests
    go test -v ./...
}

function go:clean { # remove go built binaries
    rm -rf $BIN_NAME
}

function go:lint { # lint go files in repository
    go build ./...
    go vet ./...
}

function go:fmt { # format all go files in repository
    go fmt ./...
}

function fmt { # format all files in repository
    go:fmt
    deno fmt
}

function clean { # remove all build artifacts
    go:clean
}

function default {
    go:build
}

function help {
    echo "$0 <task> <args>"
    echo "Tasks:"
    print -l ${(ok)functions} | cat -n
}

TIMEFORMAT="Task completed in %3lR"
time ${@:-default}
