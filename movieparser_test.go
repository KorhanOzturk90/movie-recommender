package main

import (
	"fmt"
	"testing"
	"github.com/aws/aws-lambda-go/events"
	"github.com/ericdaugherty/alexa-skills-kit-golang"
	"github.com/stretchr/testify/assert"
)

func TestLinkExtractMovieIdFromTitleLink(*testing.T) {
	extractedMovieId := extractMovieIdFromTitleLink("/title/tt0120586/?ref_=tt_rec_tti")
	fmt.Println("extractor result: " , extractedMovieId)
}

func TestHandler(t *testing.T) {
	m := make(map[string]string)
	m["movieName"] = "the+godfather"
	request := events.APIGatewayProxyRequest{
		QueryStringParameters: m,
	}
	Handler(request)
}

func TestAlexaHandlerNoMovieSpecified(t *testing.T) {
	intentMap := make(map[string]alexa.IntentSlot)
	intent := alexa.Intent{"movieparserIntent", "", intentMap}
	alexaRequest := alexa.Request{"", "", "", "", "", intent, "movie suggester"}
	outputSpeech := alexa.OutputSpeech{"", "", ""}
	card := alexa.Card{"", "", "", "", nil}
	alexaResponse := alexa.Response{&outputSpeech, &card, nil, nil, true}
	processAlexaIntent(&alexaRequest, &alexaResponse)
	assert.Equal(t, "Please make sure you specify the movie name based on which recommendations will be made", alexaResponse.OutputSpeech.Text, "Error message should be returned when there is no movie name!")
}

func TestAlexaHandler(t *testing.T) {
	intentMap := make(map[string]alexa.IntentSlot)
	intent := alexa.Intent{"movieparserIntent", "", intentMap}
	alexaRequest := alexa.Request{"", "", "", "", "", intent, "movie suggester"}
	outputSpeech := alexa.OutputSpeech{"", "", ""}
	card := alexa.Card{"", "", "", "", nil}
	alexaResponse := alexa.Response{&outputSpeech, &card, nil, nil, true}
	processAlexaIntent(&alexaRequest, &alexaResponse)
	assert.Equal(t, "Please make sure you specify the movie name based on which recommendations will be made", alexaResponse.OutputSpeech.Text, "Error message should be returned when there is no movie name!")
}

func TestRedisClient(t *testing.T) {
	redisClient()
}
