package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/ericdaugherty/alexa-skills-kit-golang"
	"github.com/go-redis/redis"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const cardTitle = "movieSuggester"
const Recommended_movie_intent = "movieparserIntent"
const Recommended_streaming_intent = "topstreamingIntent"

var (
	alexaMetaData = &alexa.Alexa{ApplicationID: "amzn1.ask.skill.27d938e4-00fb-462b-83fe-633ddcf27386", RequestHandler: &movieparser{}, IgnoreApplicationID: true, IgnoreTimestamp: true}
)

type omdbInfo struct {
	Title  string
	ImdbID string
	Type   string
	Year   string
	Plot   string
	Metascore string
	ImdbRating string
}

type movie struct {
	Id          int64
	Title       string
	Url         string
	TomatoScore int
}

func main() {

	lambda.Start(Handle)

}

type movieparser struct {
}

func Handle(ctx context.Context, requestEnv *alexa.RequestEnvelope) (interface{}, error) {
	return alexaMetaData.ProcessRequest(ctx, requestEnv)
}

func (h *movieparser) OnSessionStarted(ctx context.Context, request *alexa.Request, session *alexa.Session, ctx_ptr *alexa.Context, response *alexa.Response) error {

	log.Printf("OnSessionStarted requestId=%s, sessionId=%s", request.RequestID, session.SessionID)
	return nil
}

func (h *movieparser) OnLaunch(ctx context.Context, request *alexa.Request, session *alexa.Session, ctx_ptr *alexa.Context, response *alexa.Response) error {
	speechText := "Welcome to Movie Suggester. You can get movie recommendations by saying a movie name you like."

	log.Printf("OnLaunch requestId=%s, sessionId=%s", request.RequestID, session.SessionID)

	response.SetSimpleCard(cardTitle, speechText)
	response.SetOutputText(speechText)
	response.SetRepromptText(speechText)

	response.ShouldSessionEnd = false

	return nil
}

func (h *movieparser) OnIntent(ctx context.Context, request *alexa.Request, session *alexa.Session, ctx_ptr *alexa.Context, response *alexa.Response) error {
	log.Printf("OnIntent requestId=%s, sessionId=%s, intent=%s", request.RequestID, session.SessionID, request.Intent.Name)
	return processAlexaIntent(request, response)
}

func processAlexaIntent(request *alexa.Request, response *alexa.Response) error {
	filmToSearch := request.Intent.Slots["movie"].Value

	switch request.Intent.Name {
	case Recommended_streaming_intent:
		topStreamingMovies := parseAllStreamingMovies()
		var responseText = "The highly rated top 5 movies streaming are "
		for _, movie := range topStreamingMovies {
			responseText += movie.Title + ", "
		}
		response.SetSimpleCard(cardTitle, "stream")
		response.SetOutputText(responseText)

	case Recommended_movie_intent:
		log.Printf("movieparser Intent triggered with %s", filmToSearch)
		if len(filmToSearch) == 0 {
			response.SetOutputText("Please make sure you specify the movie name based on which recommendations will be made")
		} else {
			movieId := getImdbIdFromMovieName(filmToSearch)
			recommendedMoviesIdList := readImdbPageSource("https://www.imdb.com/title/" + movieId)

			var recommendedMoviesDetailedList [5]omdbInfo
			for ind, element := range recommendedMoviesIdList {
				if element != "" {
					recommendedMoviesDetailedList[ind] = getOmdbDetailedInfoFromId(element)
				}
			}
			response.SetSimpleCard(cardTitle, recommendedMoviesDetailedList[0].Title)
			response.SetOutputText("If you enjoyed " + filmToSearch + " you might also enjoy watching " +
				recommendedMoviesDetailedList[0].Title + " with a IMDB rating of " + recommendedMoviesDetailedList[0].ImdbRating + ", " +
				recommendedMoviesDetailedList[1].Title + " with a IMDB rating of " + recommendedMoviesDetailedList[1].ImdbRating + ", " +
				recommendedMoviesDetailedList[2].Title + " with a IMDB rating of " + recommendedMoviesDetailedList[2].ImdbRating + " and" +
				recommendedMoviesDetailedList[3].Title + " with a IMDB rating of " + recommendedMoviesDetailedList[3].ImdbRating)

			return nil
		}

	case "AMAZON.HelpIntent":
		log.Println("AMAZON.HelpIntent triggered")
		speechText := "You can use this app to get movie recommendations similar to the ones you like. The data for the recommended movies come from " +
			"real people's input on various websites like IMDB. Would you like to tell a movie to get similar ones?"

		response.SetSimpleCard(cardTitle, speechText)
		response.SetOutputText(speechText)
		response.SetRepromptText(speechText)
		response.ShouldSessionEnd = false

	case "AMAZON.StopIntent":
		log.Println("AMAZON.StopIntent triggered")
		response.ShouldSessionEnd = true

	case "AMAZON.CancelIntent":
		log.Println("AMAZON.CancelIntent triggered")
		response.ShouldSessionEnd = true

	default:
		return errors.New("Invalid Intent")
	}

	return nil
}

func (h *movieparser) OnSessionEnded(ctx context.Context, request *alexa.Request, session *alexa.Session, ctx_ptr *alexa.Context, response *alexa.Response) error {

	log.Printf("OnSessionEnded requestId=%s, sessionId=%s", request.RequestID, session.SessionID)
	return nil
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
	var omdbFilmInfo omdbInfo
	url := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&i=%s", os.Getenv("API_KEY"), movieID)

	cacheSet, err := strconv.ParseBool(os.Getenv("CACHE_ENABLED"))
	if err != nil {
		log.Printf("Invalid env var CACHE_ENABLED: %s", os.Getenv("CACHE_ENABLED"))
	}
	if cacheSet {
		redisCache := redisClient()
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
	url := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&t=%s", os.Getenv("API_KEY"), strings.Replace(movieName, " ", "+", -1))
	omdbFilmInfo := getOmdbMovieInfo(url)
	fmt.Printf("OMDB movie id for movieName %s -> %s \n", movieName, omdbFilmInfo.ImdbID)
	return omdbFilmInfo.ImdbID
}

func readImdbPageSource(url string) [5]string {
	resp, _ := http.Get(url)

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
			fmt.Printf("Error Token -> %s", z.Token().String())
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
							if count == 5 {
								return recommendedMoviesIdsList
							}
							break
						}
						if strings.Contains(a.Val, "discover-watch") {
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

func parseAllStreamingMovies() []movie {

	//jsonData, err := ioutil.ReadFile("movie_stream_list.json")
	//check(err)

	jsonData := readStreamSourceFile()

	var streamingMovies []movie
	err := json.Unmarshal(jsonData, &streamingMovies)
	if err != nil {
		log.Println(err)
	}

	sort.Slice(streamingMovies, func(i, j int) bool {
		return streamingMovies[i].TomatoScore > streamingMovies[j].TomatoScore
	})

	for _, movie := range streamingMovies {
		fmt.Printf("popular movie: %v - %v\n", movie.Title, movie.TomatoScore)
	}

	return streamingMovies[:5]
}

func readStreamSourceFile() []byte {
	svc := s3.New(session.New())
	input := &s3.GetObjectInput{
		Bucket: aws.String("streamed-movies"),
		Key:    aws.String("movie_stream_list.json"),
	}

	result, err := svc.GetObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				fmt.Println(s3.ErrCodeNoSuchKey, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil
	}

	fmt.Println(result)

	if b, err := ioutil.ReadAll(result.Body); err == nil {
		return b
	}
	return nil
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func redisClient() *redis.Client {
	redisDB := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL") + ":6379",
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0, // use default DB
	})

	_, err := redisDB.Ping().Result()

	if err != nil {
		fmt.Println(err)
	}

	return redisDB
}
