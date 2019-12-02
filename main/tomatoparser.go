package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
)

type movie struct {
	Id          int64
	Title       string
	Url         string
	TomatoScore int
}

func parseAllStreamingMovies() []movie {

	jsonData, err := ioutil.ReadFile("../movie_stream_list.json")
	check(err)

	var streamingMovies []movie
	err = json.Unmarshal(jsonData, &streamingMovies)
	if err != nil {
		log.Println(err)
	}

	sort.Slice(streamingMovies, func(i, j int) bool {
		return streamingMovies[i].TomatoScore > streamingMovies[j].TomatoScore
	})

	for _, movie := range streamingMovies {
		fmt.Printf("popular movie: %v - %v\n", movie.Title, movie.TomatoScore)
	}

	return streamingMovies
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
