package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

const (
	hasNone = 0
	hasLink = 1 << iota
	hasText
)

type glinksLink struct {
	DB   string `json:"db"`
	ID   string `json:"id"`
	Link string `json:"link,omitempty"`
	Text string `json:"text,omitempty"`
	Flag int    `json:",omitempty"`
}

func createGlinksLink(db, id, link, text string) (item glinksLink) {
	item.DB = db
	item.ID = id

	if len(link) > 0 {
		item.Link = link
		item.Flag |= hasLink
	}

	if len(text) > 0 {
		item.Text = text
		item.Flag |= hasText
	}

	return item
}

func (g glinksLink) HTML() []string {
	var ret []string

	if g.Flag&hasLink != 0 {
		if len(g.Link) > 0 {
			ret = append(ret, formatLink(g.DB, g.ID, g.Link))
		}
	}

	if g.Flag&hasText != 0 {
		if len(g.Text) > 0 {
			ret = append(ret, formatText(g.DB, g.ID, g.Text))
		}
	}

	return ret
}

func formatText(db, id, text string) string {
	return fmt.Sprintf(
		"<tr><td># %s</td><td>%s</td><td>%s</td></tr>",
		db, id, text,
	)
}

func formatLink(db, id, link string) string {
	return fmt.Sprintf(
		"<tr><td>%s</td><td>%s</td><td><a href=\"%s\">%s</a></td></tr>",
		db, id, link, link,
	)
}

func (g glinksLink) TSV() []string {
	var ret []string

	if g.Flag&hasLink != 0 {
		ret = append(ret, strings.Join([]string{g.DB, g.ID, g.Link}, "\t"))
	}

	if g.Flag&hasText != 0 {
		ret = append(ret, strings.Join([]string{"# " + g.DB, g.ID, g.Text}, "\t"))
	}

	return ret
}

type glinks struct {
	ID        string
	Links     []glinksLink
	UpdatedAt time.Time
}

func (g glinks) HTML() string {
	var body []string

	for _, item := range g.Links {
		body = append(body, item.HTML()...)
	}

	sort.Strings(body)

	html := fmt.Sprintf(
		"<table style=\"font-size: 0.8rem;\">"+
			"<thead style=\"text-align: left;\">"+
			"<tr><th>Database</th><th>ID</th><th>Description</th></tr></thead>"+
			"<tbody>"+
			"%s"+
			"</tbody>"+
			"</table>",
		strings.Join(body, "\n"),
	)

	return html
}

func (g glinks) TSV() string {
	var body []string

	for _, item := range g.Links {
		body = append(body, item.TSV()...)
	}

	sort.Strings(body)

	return fmt.Sprintf("%s\n//\n", strings.Join(body, "\n"))
}

type glinksCompatible interface {
	ToGlinks() []glinksLink
}

func getGlinks(ids []string) (ret []glinks, err error) {
	list, err := getUniprot(ids)

	if err != nil {
		log.Printf("Failed to get UniProt entries: %s", err)
	}

	var keggIDs []string

	for _, item := range list {
		for _, dbReference := range item.DbReference {
			if dbReference.Type == "KEGG" {
				keggIDs = append(keggIDs, dbReference.ID)
			}
		}
	}

	_, err = getLinkDB(keggIDs)

	if err != nil {
		log.Printf("Failed to get LinkDB entries: %s", err)
	}

	for _, item := range list {
		ret = append(ret, item.ToGlinks())
	}

	return ret, nil
}
