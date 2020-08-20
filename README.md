# Movie Suggester

This is a simple movie suggestion engine that recommends movies from IMDB similar to the one specified. The aim of the project is to teach myself to write code in Golang


[![CircleCI](https://circleci.com/gh/KorhanOzturk90/movie-suggester.svg?style=svg)](https://circleci.com/gh/KorhanOzturk90/movie-suggester)

## Alexa Skill Link
The algorithm can be used as an alexa skill from the following link:

https://alexa-skills.amazon.com/apis/custom/skills/amzn1.ask.skill.27d938e4-00fb-462b-83fe-633ddcf27386/launch

### Building

```
go get
GOOS=linux go build -o movieparser movieparser.go
```

## Running the tests

`go test`
