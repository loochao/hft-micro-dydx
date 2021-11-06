#!/usr/bin/env bash

cd ../../

go build -o "/usr/local/bin/dydx-vc" ./applications/dydx-vc

chmod 777  /usr/local/bin/dydx-vc

