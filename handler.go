package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo"
)

func init() {
	e.GET("/favicon.ico", func(c echo.Context) (err error) {
		return c.NoContent(http.StatusNotFound)
	})
	e.GET("/:query", handler)
}

func queryToUniprot(query string) ([]string, error) {
	for k, v := range mappings {
		var item Mapping

		if err := v.One("ID", query, &item); err != nil {
			if k == "RefSeq_NT" || k == "RefSeq" {
				for i := 0; i < 10; i++ {
					version := fmt.Sprintf("%s.%d", query, i)
					v.One("ID", version, &item)
				}
			}
		} else {
			return item.Relations, nil
		}
	}

	return nil, errConversionFailed
}

func handler(c echo.Context) error {
	queries := strings.Split(c.Param("query"), ",")
	format := c.QueryParam("format")

	var converted []string

	for _, query := range queries {
		ids, err := queryToUniprot(query)

		if err != nil {
			converted = append(converted, query)
		} else {
			converted = append(converted, ids...)
		}
	}

	list, err := getGlinks(converted)

	if err != nil {
		return err
	}

	if format == "json" {
		ret := make(map[string][]glinksLink)

		for _, item := range list {
			for i := range item.Links {
				item.Links[i].Flag = hasNone
			}
			ret[item.ID] = item.Links
		}

		return c.JSON(http.StatusOK, ret)
	}

	response := c.Response()

	response.Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	response.WriteHeader(http.StatusOK)

	for _, item := range list {
		if _, err := response.Write([]byte(item.HTML())); err != nil {
			return err
		}

		c.Response().Flush()
	}

	c.Response().Flush()

	return nil
}
