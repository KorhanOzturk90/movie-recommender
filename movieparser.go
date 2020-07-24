package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/ericdaugherty/alexa-skills-kit-golang"
	"github.com/go-redis/redis"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const cardTitle = "movieSuggester"
const Recommended_movie_intent = "movieparserIntent"
const Recommended_series_intent = "tvSeriesIntent"
const Recommended_streaming_intent = "topstreamingIntent"
const Recommended_genre_intent = "genreIntent"
const Movie_detail_intent = "movieDetailIntent"
const NO_GENRE_MOVIES = 5

var (
	alexaMetaData = &alexa.Alexa{ApplicationID: "amzn1.ask.skill.27d938e4-00fb-462b-83fe-633ddcf27386", RequestHandler: &movieparser{}, IgnoreApplicationID: true, IgnoreTimestamp: true}
)

type omdbInfo struct {
	Title      string
	ImdbID     string
	Type       string
	Year       string
	Plot       string
	Metascore  string
	ImdbRating string
}

type movie struct {
	Id          int64
	Title       string
	Url         string
	TomatoScore int
}

type GenreMovies struct {
	Context string `json:"@context"`
	Type    string `json:"@type"`
	About   struct {
		Type            string `json:"@type"`
		ItemListElement []struct {
			Type     string `json:"@type"`
			Position string `json:"position"`
			URL      string `json:"url"`
		} `json:"itemListElement"`
	} `json:"about"`
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

	log.Printf("OnSessionStarted session=%v, request=%v", session, request)
	return nil
}

func (h *movieparser) OnLaunch(ctx context.Context, request *alexa.Request, session *alexa.Session, ctx_ptr *alexa.Context, response *alexa.Response) error {
	speechText := "Welcome to Movie Suggester. You can get great movie or series recommendations similar to the ones you like. " +
		"You can say movies, or tv series to get started. Alternatively you can say genre to specify which type of movies you want"

	log.Printf("OnLaunch deviceId=%v, session=%v, request=%v", ctx_ptr.System.Device.DeviceID, session, request)

	response.SetSimpleCard(cardTitle, speechText)
	response.SetOutputText(speechText)
	response.SetRepromptText(speechText)

	response.ShouldSessionEnd = false

	return nil
}

func (h *movieparser) OnIntent(ctx context.Context, request *alexa.Request, session *alexa.Session, ctx_ptr *alexa.Context, response *alexa.Response) error {
	log.Printf("OnIntent userId=%s, sessionId=%s, requestId=%s", session.User.UserID, session.SessionID, request.RequestID)
	return processAlexaIntent(request, response)
}

func processAlexaIntent(request *alexa.Request, response *alexa.Response) error {

	switch request.Intent.Name {

	case Recommended_streaming_intent:
		topStreamingMovies := parseAllStreamingMovies()
		var responseText = "The highly rated top 5 movies streaming are "
		for _, movie := range topStreamingMovies {
			responseText += movie.Title + ", "
		}
		response.SetSimpleCard(cardTitle, "stream")
		response.SetOutputText(responseText)
		return nil

	case Recommended_movie_intent:
		var filmToSearch string
		if len(request.Intent.Slots["movieQuery"].Value) > 0 {
			filmToSearch = request.Intent.Slots["movieQuery"].Value
		} else {
			handleFallback(response)
			return nil
		}
		log.Printf("movieparser Intent triggered with %s", filmToSearch)
		findRecommendations(request, filmToSearch, response)

	case Recommended_series_intent:
		seriesName := request.Intent.Slots["series"].Value
		log.Printf("tvSeries Intent triggered with %s", seriesName)
		findRecommendations(request, seriesName, response)

	case Movie_detail_intent:
		movieName := request.Intent.Slots["movieName"].Value
		movieId := getImdbIdFromMovieName(movieName)
		if len(movieId) == 0 {
			log.Printf("Could not find the movie %s in omdb..", movieName)
			response.SetOutputText("Sorry, cannot find the movie " + movieName + " please make sure you use the correct name")
			response.ShouldSessionEnd = false
			return nil
		}
		selectedMovieDetails := getOmdbDetailedInfoFromId(movieId)
		log.Printf("Movie details are returned for : %s", selectedMovieDetails.Title)
		response.SetOutputText("Name a genre you'd like movie recommendations in please")
		response.ShouldSessionEnd = false
		return nil

	case Recommended_genre_intent:
		genre := request.Intent.Slots["genre"].Resolutions.ResolutionsPerAuthority[0].Values[0].Value.Name
		if request.Intent.ConfirmationStatus == "DENIED" {
			log.Printf("User confirmation denied for genre: %s", genre)
			response.SetOutputText("Name a genre you'd like movie recommendations in please")
			response.ShouldSessionEnd = false
			return nil
		}
		log.Printf("genre Intent triggered with %s", genre)
		dat := readStreamSourceFile("moviesuggester", genre+".json")

		var allMoviesForGenre GenreMovies

		err := json.Unmarshal(dat, &allMoviesForGenre)
		check(err)

		rand.Seed(time.Now().UnixNano())
		ch := make(chan omdbInfo)

		for x := 0; x < NO_GENRE_MOVIES; x++ {
			randomIndex := rand.Intn(len(allMoviesForGenre.About.ItemListElement))
			urlSelected := allMoviesForGenre.About.ItemListElement[randomIndex].URL
			r, _ := regexp.Compile("/title/([a-zA-Z0-9]+)/")
			randMovieId := r.FindStringSubmatch(urlSelected)[1]

			go func(imdbId string) {
				ch <- getOmdbDetailedInfoFromId(imdbId)
			}(randMovieId)
		}

		readRecommendationsFromTheChannel("I've found the following movies for "+genre+", ", ch, response)

	case "AMAZON.HelpIntent":
		log.Println("AMAZON.HelpIntent triggered")
		speechText := "Use this skill to get movie or tv series recommendations. You can start by saying movies, tv series or genre."

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

	case "AMAZON.FallbackIntent":
		log.Println("AMAZON.FallbackIntent triggered")
		handleFallback(response)

	default:
		log.Println("Could not match any intents")
		handleFallback(response)
	}
	return nil
}

func findRecommendations(request *alexa.Request, filmToSearch string, response *alexa.Response) {
	if request.Intent.ConfirmationStatus == "DENIED" {
		log.Printf("User confirmation denied for %s", filmToSearch)
		if request.Intent.Name == Recommended_movie_intent {
			response.SetOutputText("Name a movie you enjoyed please")
		} else {
			response.SetOutputText("Name a tv series you enjoyed please")
		}
		response.ShouldSessionEnd = false
	} else {
		movieId := getImdbIdFromMovieName(filmToSearch)
		if len(movieId) == 0 {
			log.Printf("Could not find the movie %s in omdb..", filmToSearch)
			response.SetOutputText("Sorry, cannot find the movie " + filmToSearch + " please make sure you use the correct name")
			response.ShouldSessionEnd = false
			return
		}

		recommendedMoviesIdList := readImdbPageSource("https://www.imdb.com/title/" + movieId)
		ch := make(chan omdbInfo)
		for _, element := range recommendedMoviesIdList {
			go func(imdbId string) {
				ch <- getOmdbDetailedInfoFromId(imdbId)
			}(element)
		}
		readRecommendationsFromTheChannel("If you enjoyed "+filmToSearch+" you might also enjoy watching ", ch, response)
	}
}

func readRecommendationsFromTheChannel(initialText string, ch chan omdbInfo, response *alexa.Response) {
	var responseText strings.Builder
	responseText.WriteString(initialText)
	for x := 0; x < 5; x++ {
		recommendedMovieDetail := <-ch
		responseText.WriteString(recommendedMovieDetail.Title)
		responseText.WriteString(" rated ")
		responseText.WriteString(recommendedMovieDetail.ImdbRating + ", ")
	}
	response.SetSimpleCard("Movie Suggestions", responseText.String())
	response.SetOutputText(responseText.String())
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
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		check(err)

		jsonErr := json.Unmarshal(contents, &ombdInfo)
		check(jsonErr)
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
			check(err)

			//Put the movie in the cache
			errCache := redisCache.Set(movieID, string(moviePayloadInJson), 0).Err()
			check(errCache)

		} else if err != nil {
			log.Printf("An error occurred while trying to connect to the cache: %s\n", err)
		} else {
			fmt.Printf("movie with ID %s found in the cache -> %s\n", movieID, cachedMovie)
			jsonErr := json.Unmarshal(cachedMovie, &omdbFilmInfo)
			check(jsonErr)
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

func check(e error) {
	if e != nil {
		fmt.Println(e)
	}
}

func handleFallback(response *alexa.Response) {
	invalidText := "Sorry, I couldn't find that"
	response.SetSimpleCard(cardTitle, invalidText)
	response.SetOutputText(invalidText)
	response.SetRepromptText("Please try with a different movie or series name")
	response.ShouldSessionEnd = false
}

func redisClient() *redis.Client {
	redisDB := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL") + ":6379",
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0, // use default DB
	})

	_, err := redisDB.Ping().Result()

	if err != nil {
		check(err)
	}

	return redisDB
}
