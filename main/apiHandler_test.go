package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHandler(t *testing.T) {
	m := make(map[string]string)
	m["movieName"] = "the+mist"
	request := events.APIGatewayProxyRequest{
		QueryStringParameters: m,
	}
	response, err := APIRequestHandler(request)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("response %v", response)
		assert.Equal(t, response.StatusCode, 200, "Unexpected Http Response")
	}
}
