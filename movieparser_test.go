package main

import (
	"fmt"
	"github.com/ericdaugherty/alexa-skills-kit-golang"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestLinkExtractMovieIdFromTitleLink(*testing.T) {
	extractedMovieId := extractMovieIdFromTitleLink("/title/tt0120586/?ref_=tt_rec_tti")
	fmt.Println("extractor result: ", extractedMovieId)
}

func TestAlexaHandlerWithMultipleRequests(t *testing.T) {
	movie1 := "Lion King"
	movie2 := "American Sniper"

	requestEnv := createAlexaRequestEnvelope(movie1)
	requestEnv2 := createAlexaRequestEnvelope(movie2)

	println("Getting recommendations for " + movie1)
	sendAlexaCommand(&requestEnv)
	time.Sleep(time.Second * 1)

	println("\n\n\nGetting recommendations for " + movie2)
	response := sendAlexaCommand(&requestEnv2)

	assert.Contains(t, response.Response.OutputSpeech.Text, "If you enjoyed "+movie2+" you might also enjoy watching ",
		"Expected response not returned")
}

func TestAlexaHandlerWithNonExistingMovie(t *testing.T) {
	movie1 := "this movie don't exist"

	requestEnv := createAlexaRequestEnvelope(movie1)

	println("Getting recommendations for " + movie1)
	response := sendAlexaCommand(&requestEnv)
	assert.Equal(t, "Sorry, cannot find the movie "+movie1+" please make sure you use the correct name",
		response.Response.OutputSpeech.Text, "Unknown movie message error")
	assert.Equal(t, false, response.Response.ShouldSessionEnd)
}

func createAlexaRequestEnvelope(movieName string) alexa.RequestEnvelope {
	intentSlots := make(map[string]alexa.IntentSlot)
	intentSlots["movieName"] = alexa.IntentSlot{
		Name:  "movieName",
		Value: movieName,
	}
	intent := alexa.Intent{
		Name:  Recommended_movie_intent,
		Slots: intentSlots,
	}

	request := alexa.Request{
		Intent: intent,
		Type:   "IntentRequest",
	}

	att := alexa.Session{}.Attributes

	session := &alexa.Session{
		SessionID:  "amzn1.echo-api.session.ee60a355-25ce-463f-a1d2-f3cd1c98a575",
		Attributes: att,
	}
	session.User.UserID = "amzn1.ask.account.AEEEJ7PUOPEKQR3AIGJOAFU5W4K273VCIFCJTKPOQ3CKURU2PUWUABCYYVKCKK466ASTAWEGF2X7S57I3E7RGBDTLBLF3HRPZBXSHDINBCXLRXURY6DNNLZXWE5F6LRSJYQ4KGHWF5KSBPXP4HBJAKRHKU32H3CCB4XPCIJOJAIHRB76PZR3GXW3JYFTBSB4MXTFW54OECM6GBA"

	requestEnv := alexa.RequestEnvelope{
		Request: &request,
		Session: session,
	}
	return requestEnv
}

func TestAlexaTopStreamingMoviesHandler(t *testing.T) {

	t.Skip("Skipping movie parser testing for now.")
	intentSlots := make(map[string]alexa.IntentSlot)

	intent := alexa.Intent{
		Name:  Recommended_streaming_intent,
		Slots: intentSlots,
	}
	request := alexa.Request{
		Intent: intent,
		Type:   "IntentRequest",
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

	sendAlexaCommand(&requestEnv)
}

func sendAlexaCommand(requestEnv *alexa.RequestEnvelope) *alexa.ResponseEnvelope {

	response, err := alexaMetaData.ProcessRequest(nil, requestEnv)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("response: %v\n", response.Response.OutputSpeech.Text)
	}
	return response
}

func TestExtractMovieTitleFromLinkRegex(t *testing.T) {
	extractedTitle := extractMovieIdFromTitleLink("/title/tt0071562/?ref_=tt_sims_tti")
	assert.Equal(t, "tt0071562", extractedTitle, "Unsuccessful movie title retrieval from link")
}

func TestRedisClient(t *testing.T) {
	redisClient()
}
