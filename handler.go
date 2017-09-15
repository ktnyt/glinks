package main

import (
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

func handler(c echo.Context) error {
	queries := strings.Split(c.Param("query"), ",")

	var converted []string

	for _, query := range queries {
		ids, err := findMapping(query)

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

	if c.Request().Header.Get("Accept") == "application/json" {
		var out []glinksOut

		for _, item := range list {
			for i := range item.Links {
				item.Links[i].Flag = hasNone
			}

			out = append(out, glinksOut{
				Uniprot: item.ID,
				Results: item.Links,
			})
		}

		return c.JSON(http.StatusOK, out)
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
