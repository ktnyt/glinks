package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/asdine/storm"
)

// Mapping provides a mapping from some ID to a list of UniProt IDs
type Mapping struct {
	ID        string `storm:"id"`
	Relations []string
}

func findMapping(query string) ([]string, error) {
	tmp := strings.Split(query, ":")

	if len(tmp) > 1 {
		k := tmp[0]
		q := strings.Join(tmp[1:] ,":")

		if v, ok := mappings[k]; ok {
			var item Mapping

			if err := v.One("ID", q, &item); err == nil {
				return item.Relations, nil
			}
		}
	}

  for k, v := range mappings {
    var item Mapping

    if err := v.One("ID", query, &item); err != nil {
      if k == "RefSeq_NT" || k == "RefSeq" {
        for i := 0; i < 10; i++ {
          version := fmt.Sprintf("%s.%d", query, i)

          if err = v.One("ID", version, &item); err == nil {
						return item.Relations, nil
					}
				}
      }
    } else {
      return item.Relations, nil
    }
  }

  return nil, errConversionFailed
}

func createMappings() map[string]*storm.DB {
	mappingPath := os.Getenv("MAPPING_PATH")

	if len(mappingPath) == 0 {
		mappingPath = "mappings"
	}

	files, err := ioutil.ReadDir(mappingPath)

	if err != nil {
		log.Fatal(err)
	}

	mappings := make(map[string]*storm.DB)

	for _, file := range files {
		name := file.Name()

		f := strings.Split(name, ".")

		if f[len(f)-1] != "db" {
			continue
		}

		log.Printf("Opening %s/%s", mappingPath, name)

		db, err := storm.Open(fmt.Sprintf("%s/%s", mappingPath, name))

		if err != nil {
			log.Fatal(err)
		}

		mappings[name[:len(name)-3]] = db
	}

	return mappings
}
