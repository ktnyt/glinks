package main

import "errors"

var (
	errConversionFailed  = errors.New("query could not be converted to uniprot")
	errTimestampInvalid  = errors.New("cache timestampe was too old")
	errDBHostNotDefined  = errors.New("host for given database was not found")
	errHTTPGetClientErr  = errors.New("http get failed with client error")
	errHTTPGetServerErr  = errors.New("http get failed with server error")
	errHTTPGetUnknownErr = errors.New("http get failed for some unknown error")
)
