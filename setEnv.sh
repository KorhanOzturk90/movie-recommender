#!/usr/bin/env bash

export $(grep -v '^#' dev.env | xargs -0)
# for linux
# export $(grep -v '^#' .env | xargs -d '\n')