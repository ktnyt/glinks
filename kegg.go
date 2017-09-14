package main

import (
	"fmt"
	"strings"
)

type keggLink struct {
	ID          string
	Domain      string
	Description string
}

func (k keggLink) ToGlinks() []glinksLink {
	link, _ := getDBHost("KEGG")
	link = strings.Replace(link, ":id", k.ID, -1)

	item := createGlinksLink(fmt.Sprintf("KEGG_%s", k.Domain), k.ID, link, "")

	if len(k.Description) > 0 {
		item.Text = k.Description
		item.Flag |= hasText
	}

	return []glinksLink{item}
}
