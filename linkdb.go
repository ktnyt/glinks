package main

import (
	"log"
	"strings"
	"time"
)

var domainMap = map[string]string{
	"path": "PATHWAY",
	"ds":   "DISEASE",
	"ko":   "ORTHOLOGY",
	"br":   "BRITE",
}

type linkDB struct {
	ID        string
	Links     []keggLink
	UpdatedAt time.Time
}

func (l linkDB) GetOrthology() string {
	for _, link := range l.Links {
		if link.Domain == "ORTHOLOGY" {
			return link.ID
		}
	}
	return ""
}

func (l linkDB) SaveCache() error {
	l.UpdatedAt = time.Now()
	return db.Set("LinkDB", l.ID, &l)
}

func linkDBLoadCache(id string) (item linkDB, err error) {
	if err = db.Get("LinkDB", id, &item); err != nil {
		return item, err
	}

	if !validTimestamp(item.UpdatedAt) {
		return item, errTimestampInvalid
	}

	return item, nil
}

func (l linkDB) ToGlinks() (list []glinksLink) {
	for _, link := range l.Links {
		list = append(list, link.ToGlinks()...)
	}
	return list
}

func fetchLinkDB(list []string) (ret []linkDB, err error) {
	result, err := fetchList("http://rest.genome.jp/link/", list, "+", 4000)

	if err != nil {
		return nil, err
	}

	lines := strings.Split(result, "\n")

	var subject linkDB

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		id, tmp, _ := splitThree(line, "\t")
		domain, link := splitTwo(tmp, ":")

		if subject.ID != id {
			if len(subject.ID) > 0 {
				ret = append(ret, subject)
			}

			subject = linkDB{
				ID: id,
				Links: []keggLink{
					keggLink{
						ID:     id,
						Domain: "GENE",
					},
				},
			}
		}

		if val, ok := domainMap[domain]; ok {
			subject.Links = append(subject.Links, keggLink{
				ID:     link,
				Domain: val,
			})
		}
	}

	ret = append(ret, subject)

	return ret, nil
}

func getLinkDB(ids []string) ([]linkDB, error) {
	var cached []linkDB
	var missed []string

	for _, id := range ids {
		item, err := linkDBLoadCache(id)

		if err != nil {
			log.Printf("Failed to load LinkDB cache for %s: %s", id, err)
			missed = append(missed, id)
		} else {
			cached = append(cached, item)
		}
	}

	if len(missed) > 0 {
		list, err := fetchLinkDB(missed)

		if err != nil {
			log.Printf("Failed to fetch from LinkDB: %s", err)
		}

		for _, item := range list {
			if err := item.SaveCache(); err != nil {
				log.Printf("Failed to save LinkDB cache for %s: %s", item.ID, err)
			}
			cached = append(cached, item)
		}
	}

	return cached, nil
}
