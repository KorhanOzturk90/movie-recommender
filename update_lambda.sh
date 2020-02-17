#!/usr/bin/env bash
GOFILE=${1:-"movieparser"}
PROFILE=${2:-"default"}


GOOS=linux go build -a -v -o ${GOFILE}
zip movie_deployment.zip ${GOFILE}
aws --profile ${PROFILE} lambda update-function-code --function-name movieRecommender --zip-file fileb://movie_deployment.zip --region eu-west-1