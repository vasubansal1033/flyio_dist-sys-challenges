#!/bin/bash

SCRIPT_DIR=$(pwd)/bin
MAELSTROM_PATH=../maelstrom

mkdir -p $SCRIPT_DIR
go build -o $SCRIPT_DIR/main

"$MAELSTROM_PATH/maelstrom" test -w g-counter --bin $SCRIPT_DIR/main --node-count 3 --rate 100 --time-limit 20 --nemesis partition