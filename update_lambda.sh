#!/usr/bin/env bash
GOFILE=${1:-"movieparser"}


GOOS=linux go build -o ${GOFILE} ${GOFILE}.go
zip movie_deployment.zip ${GOFILE}
aws lambda update-function-code --function-name movieRecommender --zip-file fileb://movie_deployment.zip --region eu-west-1