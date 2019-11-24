package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/ericdaugherty/alexa-skills-kit-golang"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLinkExtractMovieIdFromTitleLink(*testing.T) {
	extractedMovieId := extractMovieIdFromTitleLink("/title/tt0120586/?ref_=tt_rec_tti")
	fmt.Println("extractor result: ", extractedMovieId)
}

func TestAlexaHandler(t *testing.T) {

	intentSlots:= make(map[string]alexa.IntentSlot)
	intentSlots["movie"] = alexa.IntentSlot{
		Name:"movie",
		Value:"Shrek",
	}
	intent := alexa.Intent{
		Name: "movieparserIntent",
		Slots: intentSlots,
	}

	request := alexa.Request{
		Intent: intent,
		Type: "IntentRequest",
	}

	att := alexa.Session{}.Attributes

	session := &alexa.Session{
		SessionID:  "testId",
		Attributes: att,
	}

	requestEnv := alexa.RequestEnvelope{
		Request: &request,
		Session: session,

	}

	response, err := Handle(nil, &requestEnv)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("response %v", response)
	}
}

func TestHandler(t *testing.T) {
	m := make(map[string]string)
	m["movieName"] = "the+mist"
	request := events.APIGatewayProxyRequest{
		QueryStringParameters: m,
	}
	response, err := Handler(request)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("response %v", response)
		assert.Equal(t, response.StatusCode, 200, "Unexpected Http Response")
	}
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

func TestExtractMovieTitleFromLinkRegex(t *testing.T) {
	extractedTitle := extractMovieIdFromTitleLink("/title/tt0071562/?ref_=tt_sims_tti")
	assert.Equal(t, "tt0071562", extractedTitle, "Unsuccessful movie title retrieval from link")
}

func TestRedisClient(t *testing.T) {
	redisClient()
}
