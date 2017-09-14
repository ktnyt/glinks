package main

import (
	"bufio"
	"log"
	"net/http"
	"sort"
	"strings"
)

type keggOrthology struct {
	ID          string `storm:"id"`
	Description string
	Links       []keggLink
}

func (k keggOrthology) getDescription(id string, domain string) string {
	if i := sort.Search(len(k.Links), func(i int) bool {
		return k.Links[i].ID == id && k.Links[i].Domain == domain
	}); i < len(k.Links) {
		return k.Links[i].Description
	}
	return ""
}

func parseKeggOrthology(entry string) (item keggOrthology) {
	lines := strings.Split(entry, "\n")

	var links []keggLink

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		fields := filterEmpty(strings.Split(line, " "))

		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "ENTRY":
			item.ID = fields[1]

		case "DEFINITION":
			item.Description = strings.Join(fields[1:], " ")

		case "PATHWAY":
			links = append(links, keggLink{
				ID:          fields[1],
				Domain:      fields[0],
				Description: strings.Join(fields[2:], " "),
			})

			for j := 1; len(lines[i+j]) != len(strings.TrimSpace(lines[i+j])); j++ {
				line = lines[i+j]
				fields = filterEmpty(strings.Split(line, " "))
				links = append(links, keggLink{
					ID:          fields[0],
					Domain:      "PATHWAY",
					Description: strings.Join(fields[1:], " "),
				})
			}

		case "DISEASE":
			links = append(links, keggLink{
				ID:          fields[1],
				Domain:      fields[0],
				Description: strings.Join(fields[2:], " "),
			})

			for j := 1; len(lines[i+j]) != len(strings.TrimSpace(lines[i+j])); j++ {
				line = lines[i+j]
				fields = filterEmpty(strings.Split(line, " "))
				links = append(links, keggLink{
					ID:          fields[1],
					Domain:      "DISEASE",
					Description: strings.Join(fields[2:], " "),
				})
			}
		}
	}

	item.Links = links

	return item
}

func init() {
	log.Println("Update KEGG Orthology")

	var tmp keggOrthology

	err := db.Get("KeggOrthology", "K22128", &tmp)

	if err == nil && tmp.ID == "K22128" {
		return
	}

	if err := updateKeggOrthology(); err != nil {
		log.Fatal(err)
	}
}

func updateKeggOrthology() error {
	res, err := http.Get("http://rest.kegg.jp/list/orthology")

	if err != nil {
		return err
	}

	defer res.Body.Close()

	scanner := bufio.NewScanner(res.Body)

	var ids []string

	for scanner.Scan() {
		line := scanner.Text()

		ko, _ := splitTwo(line, "\t")

		_, id := splitTwo(ko, ":")

		ids = append(ids, id)
	}

	result, err := fetchList("http://togows.org/entry/kegg-orthology/", ids, ",", 2000)

	if err != nil {
		return err
	}

	list := strings.Split(result, "///")

	list = list[:len(list)-1]

	for _, item := range list {
		ko := parseKeggOrthology(item)

		db.Set("KeggOrthology", ko.ID, &ko)
	}

	return nil
}
