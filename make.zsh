#!/bin/zsh

BIN_NAME="szmaterlok"
BIN_PATH="./cmd"
WATCH_SLEEP=3

function go:watch { # watch for changes and rebuild go binaries
    go:build
    ./$BIN_NAME &
    binpid=$!

    max=0
    while true; do
        # find files newer than built binary
        files=( $(find . -newer $BIN_NAME -and \( \
            -name '*.go' \
            -or -name '*.mod' \
            -or -name '*.sum' \
            -or -name '*.json' \
            -or -name '*.js' \
            -or -name '*.html' \
            -or -name '*.css' \)) )

        # if there are any files newer than built binary
        # rebuild and run szmaterlok executable again
        if (( ${#files[@]} > 0 )); then
            kill -INT $binpid
            go:build
            ./$BIN_NAME &
            binpid=$!
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
