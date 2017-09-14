package main

import (
	"bufio"
	"bytes"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func splitTwo(str string, sep string) (string, string) {
	fields := strings.Split(str, sep)
	return fields[0], fields[1]
}

func splitThree(str string, sep string) (string, string, string) {
	fields := strings.Split(str, sep)
	return fields[0], fields[1], fields[2]
}

func filterEmpty(vs []string) (vsf []string) {
	for _, v := range vs {
		if len(v) > 0 {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func validTimestamp(timestamp time.Time) bool {
	duration := time.Since(timestamp)

	return !(duration.Hours() >= 24 /* Hours */ *7 /* Days */ *2 /* Weeks */)
}

func fetchPart(url string) (string, error) {
	res, err := http.Get(url)

	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	code := res.StatusCode

	if 200 <= code && code <= 299 {
		buffer := new(bytes.Buffer)
		buffer.ReadFrom(res.Body)
		return buffer.String(), nil
	}

	if 400 <= code && code <= 499 {
		return "", errHTTPGetClientErr
	}

	if 500 <= code && code <= 599 {
		return "", errHTTPGetServerErr
	}

	return "", errHTTPGetUnknownErr
}

func fetchList(base string, list []string, sep string, limit int) (string, error) {
	i := 0

	var result string

	log.Printf("Start fetch from: %s\n", base)

	for i < len(list) {
		for j := len(list); j > i; j-- {
			args := strings.Join(list[i:j], sep)
			if len(base)+len(args) < limit {
				log.Printf("Get from %d to %d (%d) of %d\n", i+1, j, len(list[i:j]), len(list))

				str, err := fetchPart(base + args)

				if err != nil {
					return result, err
				}

				result += str

				i = j
			}
		}
	}

	return result, nil
}

func getDBHost(db string) (string, error) {
	file, err := os.Open("urls")

	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Split(line, " ")

		if fields[0] == db {
			return fields[1], nil
		}
	}

	return "", errDBHostNotDefined
}
