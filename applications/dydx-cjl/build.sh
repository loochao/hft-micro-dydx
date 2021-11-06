#!/usr/bin/env bash

cd ../../

go build -o "/usr/local/bin/dydx-cjl" ./applications/dydx-cjl

chmod 777  /usr/local/bin/dydx-cjl

