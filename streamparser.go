package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io/ioutil"
	"log"
	"sort"
)

func parseAllStreamingMovies() []movie {

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
