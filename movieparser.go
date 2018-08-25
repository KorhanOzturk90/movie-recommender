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
)

type omdbInfo struct {
	Title  string
	ImdbID string
	Type   string
	Year   string
	Plot   string
}

func main() {
	lambda.Start(Handler)
}

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	fmt.Println("query params:")
	for key, value := range request.QueryStringParameters {
		fmt.Printf("    %s: %s\n", key, value)
	}

	movieId := getImdbIdFromMovieName(request.QueryStringParameters["movieName"])
	recommendedMoviesIdList := readImdbPageSource("https://www.imdb.com/title/" + movieId)

	var recommendedMoviesDetailedList [5]omdbInfo
	for ind, element := range recommendedMoviesIdList {
		if element != "" {
			recommendedMoviesDetailedList[ind] = getOmdbDetailedInfoFromId(element)
		}
	}
	fmt.Println("Final List of Recommended Movies: ", recommendedMoviesDetailedList)

	movieListJson, err := json.Marshal(recommendedMoviesDetailedList)
	if err != nil {
		fmt.Println(err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "An error occurred while parsing movie list to JSON",
		}, err
	}

	fmt.Println("All Movies in Json ", string(movieListJson))

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
	response, err := http.Get(omdbURL)

	if err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}

		jsonErr := json.Unmarshal(contents, &ombdInfo)
		if jsonErr != nil {
			panic(jsonErr)
		}
		log.Printf("title %s and IMDB ID %s", ombdInfo.Title, ombdInfo.ImdbID)
	}
	return ombdInfo
}

func getOmdbDetailedInfoFromId(movieID string) omdbInfo {
	redisCache := redisClient()
	cachedMovie, err := redisCache.Get(movieID).Bytes()
	var omdbFilmInfo omdbInfo
	if err == redis.Nil {
		fmt.Printf("%s does not exist in the cache, caching..", movieID)
		url := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&i=%s", os.Getenv("API_KEY"), movieID)
		fmt.Printf("Getting detailed movie info from OMDB for movie id %s", movieID)
		omdbFilmInfo = getOmdbMovieInfo(url)

		moviePayloadInJson, err := json.Marshal(omdbFilmInfo)
		if err != nil {
			fmt.Println(err)
		}

		error := redisCache.Set(movieID, string(moviePayloadInJson), 0).Err()
		if error != nil {
			panic(error)
		}

	} else if err != nil {
		panic(err)
	} else {
		fmt.Printf("movie with ID %s found in the cache -> %s", movieID, cachedMovie)
		jsonErr := json.Unmarshal(cachedMovie, &omdbFilmInfo)
		if jsonErr != nil {
			panic(jsonErr)
		}
	}
	return omdbFilmInfo
}

func getImdbIdFromMovieName(movieName string) string {
	url := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&t=%s", os.Getenv("API_KEY"), movieName)
	omdbFilmInfo := getOmdbMovieInfo(url)
	fmt.Printf("OMDB movie id for movieName %s -> %s ", movieName, omdbFilmInfo.ImdbID)
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

func readFile(fileName string) io.Reader {
	f, err := os.Open(fileName)
	check(err)
	return f
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func redisClient() *redis.Client {
	redisDB := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL") + ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	pong, err := redisDB.Ping().Result()
	fmt.Println(pong, err)
	// Output: PONG <nil>

	return redisDB
}
