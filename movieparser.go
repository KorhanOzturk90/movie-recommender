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

	fmt.Println("headers:")
	for key, value := range request.Headers {
		fmt.Printf("    %s: %s\n", key, value)
	}

	movieId := getImdbIdFromMovieName(request.QueryStringParameters["movieName"])
	recommendedMoviesIdList := readImdbPageSource("https://www.imdb.com/title/" + movieId)

	fmt.Println("Final List of Recommended Movies: ", recommendedMoviesIdList)

	var recommendedMoviesDetailedList [5]string
	for ind, element := range recommendedMoviesIdList {
		if element != "" {
			recommendedMoviesDetailedList[ind] = getOmdbDetailedInfoFromId(element).Title
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

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(movieListJson),
		Headers: headers,
	}, nil
	//return recommendedMoviesIdList, nil
}

func getOmdbMovieInfo(omdbURL string) omdbInfo {
	var ombdInfo omdbInfo
	//omdbURL := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&t=%s", os.Getenv("API_KEY"), movieName)
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
			log.Fatal(jsonErr)
		}
		log.Printf("title %s and IMDB ID %s", ombdInfo.Title, ombdInfo.ImdbID)
	}
	return ombdInfo
}

func getOmdbDetailedInfoFromId(movieID string) omdbInfo{
	url := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&i=%s", os.Getenv("API_KEY"), movieID)
	omdbFilmInfo:= getOmdbMovieInfo(url)
	return omdbFilmInfo
}

func getImdbIdFromMovieName(movieName string) string {
	url := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&t=%s", os.Getenv("API_KEY"), movieName)
	omdbFilmInfo:= getOmdbMovieInfo(url)

	return omdbFilmInfo.ImdbID
}

func readImdbPageSource(url string) [6]string {
	resp, _ := http.Get(url)

	recommendedLinkList := getListOfRecommendedFilmsFromIMDBSource(resp.Body)
	fmt.Println("links list ", recommendedLinkList)
	return recommendedLinkList
}

func getListOfRecommendedFilmsFromIMDBSource(source io.Reader) [6]string {
	var foundFags bool
	var recommendedMoviesIdsList [6]string
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
						//fmt.Println("Found href:", a.Val)
						if foundFags {
							recommendedMoviesIdsList[count] = extractMovieIdFromTitleLink(a.Val)
							count += 1
							if count == 5 {
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
