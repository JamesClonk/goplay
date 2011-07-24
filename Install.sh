#!/bin/sh
set -ev

## Build
cd cmd; make install

## Clean
make clean

## Install succeeded!

