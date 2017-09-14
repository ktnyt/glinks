package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type geneOntology struct {
	ID         string `storm:"id"`
	Name       string
	Namespace  string
	Definition string
	Slim       bool
}

func (g geneOntology) ToGlinks() []glinksLink {
	_, namespace := splitTwo(g.Namespace, "_")

	link, _ := getDBHost("GO")
	link = strings.Replace(link, ":id", g.ID, -1)

	item := createGlinksLink(fmt.Sprintf("GO_%s", namespace), g.ID, link, g.Definition)

	if g.Slim {
		slim := createGlinksLink(fmt.Sprintf("GOslim_%s", namespace), g.ID, link, g.Definition)

		return []glinksLink{item, slim}
	}

	return []glinksLink{item}
}

func init() {
	updateGeneOntology()
}

func updateGeneOntology() {
	log.Println("Updating Gene Ontology")

	res, err := http.Get("http://purl.obolibrary.org/obo/go.obo")

	if err != nil {
		log.Fatal(err)
		return
	}

	defer res.Body.Close()

	scanner := bufio.NewScanner(res.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "[Term]" {
			var item geneOntology

			for scanner.Scan() {
				line = scanner.Text()

				if len(line) == 0 {
					break
				}

				prefix, body := splitTwo(line, ": ")

				switch prefix {
				case "id":
					item.ID = body
				case "name":
					item.Name = body
				case "namespace":
					item.Namespace = body
				case "def":
					item.Definition = body
				case "subset":
					item.Slim = body != "goantislim_grouping"
				}
			}

			if err := db.Set("GO", item.ID, &item); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func getGeneOntology(query string) (item geneOntology, err error) {
	err = db.Get("GO", query, &item)
	return item, err
}
