package main

import (
	"io/ioutil"
	"fmt"
	"net/http"
	"log"
	"os"
	"encoding/json"
	"golang.org/x/net/html"
	"io"
	"strings"
	"regexp"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/events"
	"context"
	"github.com/go-redis/redis"
	"errors"
	"github.com/ericdaugherty/alexa-skills-kit-golang"
	"strconv"
)

const cardTitle = "movieSuggester"
var(
 alexaMetaData = &alexa.Alexa{ApplicationID: "amzn1.ask.skill.<SKILL_ID>", RequestHandler: &movieparser{}, IgnoreApplicationID: true, IgnoreTimestamp: true}
 cacheOn = true

)

type omdbInfo struct {
	Title  string
	ImdbID string
	Type   string
	Year   string
	Plot   string
}

func main() {
	cacheSet, err := strconv.ParseBool(os.Getenv("CACHE_ENABLED"))
	cacheOn = cacheSet
	if err != nil {
		log.Printf("Invalid env var CACHE_ENABLED: %s", os.Getenv("CACHE_ENABLED"))
	}
	lambda.Start(Handle)

}

type movieparser struct {
}

func Handle(ctx context.Context, requestEnv *alexa.RequestEnvelope) (interface{}, error) {
	return alexaMetaData.ProcessRequest(ctx, requestEnv)
}

func (h *movieparser) OnSessionStarted(ctx context.Context,request *alexa.Request, session *alexa.Session, ctx_ptr *alexa.Context,response *alexa.Response) error {

	log.Printf("OnSessionStarted requestId=%s, sessionId=%s", request.RequestID, session.SessionID)
	return nil
}

func (h *movieparser) OnLaunch(ctx context.Context,request *alexa.Request, session *alexa.Session, ctx_ptr *alexa.Context,response *alexa.Response) error {
	speechText := "Welcome to Urban Dictionary App. You can look up words by saying what's the meaning of followed by your query."

	log.Printf("OnLaunch requestId=%s, sessionId=%s", request.RequestID, session.SessionID)

	response.SetSimpleCard(cardTitle, speechText)
	response.SetOutputText(speechText)
	response.SetRepromptText(speechText)

	response.ShouldSessionEnd = false

	return nil
}

func (h *movieparser) OnIntent(ctx context.Context,request *alexa.Request, session *alexa.Session, ctx_ptr *alexa.Context,response *alexa.Response) error {

	log.Printf("OnIntent requestId=%s, sessionId=%s, intent=%s", request.RequestID, session.SessionID, request.Intent.Name)
	filmToSearch := request.Intent.Slots["movie"].Value

	switch request.Intent.Name {
	case "movieparserIntent":
		log.Printf("movieparser Intent triggered with %s", filmToSearch)

		movieId := getImdbIdFromMovieName(filmToSearch)
		recommendedMoviesIdList := readImdbPageSource("https://www.imdb.com/title/" + movieId)

		var recommendedMoviesDetailedList [5]omdbInfo
		for ind, element := range recommendedMoviesIdList {
			if element != "" {
				recommendedMoviesDetailedList[ind] = getOmdbDetailedInfoFromId(element)
			}
		}
			response.SetSimpleCard(cardTitle,  recommendedMoviesDetailedList[0].Title)
			response.SetOutputText("You might enjoy watching " + recommendedMoviesDetailedList[0].Title + " if you enjoyed " + filmToSearch)

	case "AMAZON.HelpIntent":
		log.Println("AMAZON.HelpIntent triggered")
		speechText := "Use this app to learn the coolest slang words and phrases!"

		response.SetSimpleCard(cardTitle, speechText)
		response.SetOutputText(speechText)
		response.SetRepromptText(speechText)

	default:
		return errors.New("Invalid Intent")
	}

	return nil
}

func (h *movieparser) OnSessionEnded(ctx context.Context,request *alexa.Request, session *alexa.Session, ctx_ptr *alexa.Context,response *alexa.Response) error {

	log.Printf("OnSessionEnded requestId=%s, sessionId=%s", request.RequestID, session.SessionID)

	return nil
}

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	movieId := getImdbIdFromMovieName(request.QueryStringParameters["movieName"])
	recommendedMoviesIdList := readImdbPageSource("https://www.imdb.com/title/" + movieId)

	var recommendedMoviesDetailedList [5]omdbInfo
	for ind, element := range recommendedMoviesIdList {
		if element != "" {
			recommendedMoviesDetailedList[ind] = getOmdbDetailedInfoFromId(element)
		}
	}


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

func getOmdbMovieInfo(omdbURL string) omdbInfo {
	var ombdInfo omdbInfo
	fmt.Printf("will try to get omdb info for %s\n", omdbURL)
	response, err := http.Get(omdbURL)

	if err != nil {
		fmt.Print(err)
		//os.Exit(1)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Print(err)
		}

		jsonErr := json.Unmarshal(contents, &ombdInfo)
		if jsonErr != nil {
			log.Print(jsonErr)
		}
		fmt.Printf("title %s and IMDB ID %s\n", ombdInfo.Title, ombdInfo.ImdbID)
	}
	return ombdInfo
}

func getOmdbDetailedInfoFromId(movieID string) omdbInfo {
	redisCache := redisClient()
	var omdbFilmInfo omdbInfo
	url := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&i=%s", os.Getenv("API_KEY"), movieID)

	if cacheOn {
		cachedMovie, err := redisCache.Get(movieID).Bytes()
		if err == redis.Nil {
			fmt.Printf("%s does not exist in the cache, caching..\n", movieID)
			fmt.Printf("Getting detailed movie info from OMDB for movie id %s\n", movieID)
			omdbFilmInfo = getOmdbMovieInfo(url)

			moviePayloadInJson, err := json.Marshal(omdbFilmInfo)
			if err != nil {
				fmt.Println(err)
			}

			//Put the movie in the cache
			errCache := redisCache.Set(movieID, string(moviePayloadInJson), 0).Err()
			if errCache != nil {
				log.Printf("An error occurred while trying to cache the element: %s\n", errCache)
			}

		} else if err != nil {
			log.Printf("An error occurred while trying to connect to the cache: %s\n", err)
		} else {
			fmt.Printf("movie with ID %s found in the cache -> %s\n", movieID, cachedMovie)
			jsonErr := json.Unmarshal(cachedMovie, &omdbFilmInfo)
			if jsonErr != nil {
				log.Print(jsonErr)
			}
		}
	} else { //no cache
		omdbFilmInfo = getOmdbMovieInfo(url)
	}
	return omdbFilmInfo
}

func getImdbIdFromMovieName(movieName string) string {
	url := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&t=%s", os.Getenv("API_KEY"), movieName)
	omdbFilmInfo := getOmdbMovieInfo(url)
	fmt.Printf("OMDB movie id for movieName %s -> %s \n", movieName, omdbFilmInfo.ImdbID)
	return omdbFilmInfo.ImdbID
}

func readImdbPageSource(url string) [5]string {
	resp, _ := http.Get(url)
	fmt.Printf("IMDB Status code for url %s %s\n", url, resp.Status)

	recommendedLinkList := getListOfRecommendedFilmsFromIMDBSource(resp.Body)
	fmt.Println("links list ", recommendedLinkList)
	return recommendedLinkList
}

func getListOfRecommendedFilmsFromIMDBSource(source io.Reader) [5]string {
	var foundFags bool
	var recommendedMoviesIdsList [5]string
	count := 0

	z := html.NewTokenizer(source)
	for {
		currentToken := z.Next()

		switch {
		case currentToken == html.ErrorToken:
			// End of the document, we're done
			log.Printf("Error Token -> %s", z.Token().String())
			return recommendedMoviesIdsList
		case currentToken == html.StartTagToken:
			t := z.Token()

			isAnchor := t.Data == "a"
			if isAnchor {
				for _, a := range t.Attr {
					if a.Key == "href" {
						if foundFags {
							recommendedMoviesIdsList[count] = extractMovieIdFromTitleLink(a.Val)
							count += 1
							if count == 4 {
								return recommendedMoviesIdsList
							}
							break
						}
						if strings.Contains(a.Val, "recommended-for-you-faqs") {
							foundFags = true
						}
						break
					}
				}
			}
		}
	}
}

func extractMovieIdFromTitleLink(link string) string {
	r, _ := regexp.Compile("/title/([a-zA-Z0-9]+)/\\?ref")
	return r.FindStringSubmatch(link)[1]
}

func redisClient() *redis.Client {
	redisDB := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL") + ":6379",
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,  // use default DB
	})

	_, err := redisDB.Ping().Result()

	if err != nil {
		fmt.Println(err)
		cacheOn = false
	}

	return redisDB
}
