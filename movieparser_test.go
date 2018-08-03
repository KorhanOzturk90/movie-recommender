package main

import (
	"fmt"
	"testing"
	"github.com/aws/aws-lambda-go/events"
)

func TestLinkExtractMovieIdFromTitleLink(*testing.T) {
	extractedMovieId := extractMovieIdFromTitleLink("/title/tt0120586/?ref_=tt_rec_tti")
	fmt.Println("extractor result: " , extractedMovieId)
}

func TestHandler(t *testing.T) {
	m := make(map[string]string)
	m["movieName"] = "the+matrix"
	request := events.APIGatewayProxyRequest{
		QueryStringParameters: m,
	}
	Handler(nil, request)
}
