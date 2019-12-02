package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"net/http"
)

func APIRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	movieId := getImdbIdFromMovieName(request.QueryStringParameters["movieName"])
	recommendedMoviesIdList := readImdbPageSource("https://www.imdb.com/title/" + movieId)

	var recommendedMoviesDetailedList [5]omdbInfo
	for ind, element := range recommendedMoviesIdList {
		if element != "" {
			recommendedMoviesDetailedList[ind] = getOmdbDetailedInfoFromId(element)
		}
	}

	fmt.Printf("recommended movies final list %v", recommendedMoviesDetailedList)

	movieListJson, err := json.Marshal(recommendedMoviesDetailedList)
	if err != nil {
		fmt.Println(err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "An error occurred while parsing movie list to JSON",
		}, err
	}

	fmt.Println("Final List of Recommended Movies: ", string(movieListJson))
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(movieListJson),
		Headers:    headers,
	}, nil
}
