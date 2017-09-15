package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type propertyType struct {
	Type  string `xml:"type,attr"`
	Value string `xml:"value,attr"`
}

type dbReferenceType struct {
	Origin   string
	Type     string         `xml:"type,attr"`
	ID       string         `xml:"id,attr"`
	Property []propertyType `xml:"property"`
}

func (d dbReferenceType) ToGlinks() []glinksLink {
	item := glinksLink{
		DB: d.Type,
		ID: d.ID,
	}

	// Switch for type specific conversion
	switch d.Type {
	case "GO":
		item, err := getGeneOntology(d.ID)

		if err != nil {
			return make([]glinksLink, 0)
		}

		return item.ToGlinks()
	case "KEGG":
		item, err := linkDBLoadCache(d.ID)

		if err != nil {
			log.Printf("Failed to load LinkDB cache: %s", err)
			return make([]glinksLink, 0)
		}

		return item.ToGlinks()
	case "UniGene":
		org, cid := splitTwo(d.ID, ".")
		id := fmt.Sprintf("ORG=%s&CID=%s", org, cid)

		link, _ := getDBHost(d.Type)
		link = strings.Replace(link, ":id", id, -1)

		item.Link = link
		item.Flag = hasLink
	default:
		link, err := getDBHost(d.Type)
		link = strings.Replace(link, ":gene", d.Origin, -1)
		link = strings.Replace(link, ":id", d.ID, -1)

		if err != nil {
			item.Text = d.ID
			item.Flag = hasText
		} else {
			item.Link = link
			item.Flag = hasLink
		}
	}

	// Additional links
	if d.Type == "RefSeq" {
		for _, property := range d.Property {
			if property.Type == "nucleotide sequence ID" {
				link, _ := getDBHost("Nucleotide")
				link = strings.Replace(link, ":id", property.Value, -1)

				return []glinksLink{item, createGlinksLink(d.Type, property.Value, link, "")}
			}
		}
	}

	return []glinksLink{item}
}

type proteinNameGroup struct {
	Origin    string
	FullName  string   `xml:"fullName"`
	ShortName []string `xml:"shortName"`
	EcNumber  []string `xml:"ecNumber"`
}

func (p proteinNameGroup) ToGlinks() (list []glinksLink) {
	list = append(list, createGlinksLink("Full Name", p.Origin, "", p.FullName))

	for _, name := range p.ShortName {
		list = append(list, createGlinksLink("Short Name", p.Origin, "", name))
	}

	for _, ecNumber := range p.ShortName {
		list = append(list, createGlinksLink("EC Number", p.Origin, "", ecNumber))
	}

	return list
}

type proteinType struct {
	Origin          string
	RecommendedName proteinNameGroup   `xml:"recommendedName"`
	AlternativeName []proteinNameGroup `xml:"alternativeName"`
	SubmittedName   []proteinNameGroup `xml:"submittedName"`
	AllergenName    string             `xml:"allergenName"`
	BiotechName     string             `xml:"biotechName"`
	CdAntigenName   []string           `xml:"cdAntigenName"`
	InnName         []string           `xml:"innName"`
}

func (p *proteinType) AddOrigin(id string) {
	p.Origin = id
	p.RecommendedName.Origin = id

	for i := range p.AlternativeName {
		p.AlternativeName[i].Origin = id
	}

	for i := range p.SubmittedName {
		p.SubmittedName[i].Origin = id
	}
}

func (p proteinType) ToGlinks() []glinksLink {
	recommended := p.RecommendedName.ToGlinks()

	for i := range recommended {
		recommended[i].DB += " (Recommended)"
	}

	alternative := make([]glinksLink, 0)

	for _, nameGroup := range p.AlternativeName {
		alternative = append(alternative, nameGroup.ToGlinks()...)
	}

	for i := range alternative {
		alternative[i].DB += " (Alternative)"
	}

	submitted := make([]glinksLink, 0)

	for _, nameGroup := range p.SubmittedName {
		submitted = append(submitted, nameGroup.ToGlinks()...)
	}

	for i := range submitted {
		submitted[i].DB += " (Submitted)"
	}

	var links []glinksLink

	links = append(links, recommended...)
	links = append(links, alternative...)
	links = append(links, submitted...)

	return links
}

type geneNameType struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type geneType struct {
	Name []geneNameType `xml:"name"`
}

type organismNameType struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type organismType struct {
	Name        organismNameType `xml:"name"`
	DbReference dbReferenceType  `xml:"dbReference"`
	Lineage     struct {
		Taxon []string `xml:"taxon"`
	} `xml:"lineage"`
}

func (o organismType) ToGlinks() []glinksLink {
	uniprotLink, _ := getDBHost("UniProtTaxonomy")
	uniprotLink = strings.Replace(uniprotLink, ":id", o.DbReference.ID, -1)

	uniprotTaxonomy := createGlinksLink("UniProt Taxonomy", o.DbReference.ID, uniprotLink, "")

	ncbiLink, _ := getDBHost("NCBITaxonomy")
	ncbiLink = strings.Replace(ncbiLink, ":id", o.DbReference.ID, -1)

	ncbiTaxonomy := createGlinksLink("NCBI Taxonomy", o.DbReference.ID, ncbiLink, "")

	text := strings.Join(o.Lineage.Taxon, "; ")

	lineage := createGlinksLink("Lineage", o.DbReference.ID, "", text)

	return []glinksLink{uniprotTaxonomy, ncbiTaxonomy, lineage}
}

type geneLocationType struct {
	Type string `xml:"type,attr"`
	Name string `xml:"name"`
}

func (g geneLocationType) ToGlinks() []glinksLink {
	return []glinksLink{createGlinksLink("Gene Location", g.Name, "", g.Type)}
}

type citationType struct {
	DbReference []dbReferenceType `xml:"dbReference"`
}

type referenceType struct {
	Citation citationType `xml:"citation"`
}

type interactantType struct {
	IntactID string `xml:"intactId,attr"`
	ID       string `xml:"id"`
	Label    string `xml:"label"`
}

type diseaseType struct {
	ID          string          `xml:"id,attr"`
	Name        string          `xml:"name"`
	Acronym     string          `xml:"acronym"`
	Description string          `xml:"description"`
	DbReference dbReferenceType `xml:"dbReference"`
}

type commentType struct {
	Origin      string
	Type        string            `xml:"type,attr"`
	Text        string            `xml:"text"`
	Disease     []diseaseType     `xml:"disease"`
	Interactant []interactantType `xml:"interactant"`
}

func (c commentType) ToGlinks() []glinksLink {
	item := glinksLink{
		DB:   c.Type,
		Flag: hasText,
	}

	switch c.Type {
	case "interaction":
		intactA := c.Interactant[0]
		intactB := c.Interactant[1]
		item.ID = intactB.ID
		item.Text = fmt.Sprintf("%s:%s", intactA.IntactID, intactB.IntactID)
	case "disease":
		var list []glinksLink
		for _, disease := range c.Disease {
			text := fmt.Sprintf(
				"%s (%s) : %s (%s)",
				disease.Name,
				disease.Acronym,
				disease.Description,
				c.Text,
			)
			list = append(list, createGlinksLink(c.Type, disease.ID, "", text))
		}
		return list
	default:
		item.ID = c.Origin
		item.Text = c.Text
	}

	return []glinksLink{item}
}

type uniprot struct {
	ID           string            `storm:"id"`
	Accession    []string          `xml:"accession"`
	Name         []string          `xml:"name"`
	Protein      proteinType       `xml:"protein"`
	Gene         geneType          `xml:"gene"`
	Organism     organismType      `xml:"organism"`
	GeneLocation geneLocationType  `xml:"geneLocation"`
	Reference    []referenceType   `xml:"reference"`
	Comment      []commentType     `xml:"comment"`
	DbReference  []dbReferenceType `xml:"dbReference"`
	UpdatedAt    time.Time
}

func (u uniprot) SaveCache() error {
	for _, accession := range u.Accession {
		db.Set("UniProtMapping", accession, u.ID)
	}
	u.UpdatedAt = time.Now()
	return db.Set("UniProt", u.ID, &u)
}

func uniprotLoadCache(id string) (item uniprot, err error) {
	var accession string

	if err = db.Get("UniProtMapping", id, &accession); err != nil {
		if err = db.Get("UniProt", id, &item); err != nil {
			return item, err
		}
	} else {
		if err = db.Get("UniProt", accession, &item); err != nil {
			return item, err
		}
	}

	if !validTimestamp(item.UpdatedAt) {
		return item, errTimestampInvalid
	}

	return item, nil
}

func (u *uniprot) ToGlinks() glinks {
	var compatible []glinksCompatible

	u.Protein.AddOrigin(u.ID)
	compatible = append(compatible, u.Protein)

	compatible = append(compatible, u.Organism)
	compatible = append(compatible, u.GeneLocation)

	for _, comment := range u.Comment {
		comment.Origin = u.ID
		compatible = append(compatible, comment)
	}

	for _, dbReference := range u.DbReference {
		dbReference.Origin = u.ID
		compatible = append(compatible, dbReference)
	}

	links := make([]glinksLink, 0)

	for _, accession := range u.Accession {
		link, _ := getDBHost("UniProtKB-AC")
		link = strings.Replace(link, ":id", accession, -1)
		links = append(links, createGlinksLink("UniProtKB-AC", accession, link, ""))
	}

	for _, name := range u.Name {
		link, _ := getDBHost("UniProtKB-ID")
		link = strings.Replace(link, ":id", name, -1)
		links = append(links, createGlinksLink("UniProtKB-ID", name, link, ""))
	}

	for _, compatible := range compatible {
		list := compatible.ToGlinks()
		links = append(links, list...)
	}

	return glinks{
		ID:    u.ID,
		Links: links,
	}
}

type uniprotBase struct {
	Entry []uniprot `xml:"entry"`
}

func fetchUniprot(ids []string) ([]uniprot, error) {
	var res *http.Response
	var err error

	if len(ids) == 1 {
		res, err = http.Get(fmt.Sprintf("http://www.uniprot.org/uniprot/%s.xml", ids[0]))

		if err != nil {
			return nil, err
		}

		defer res.Body.Close()

		return processUniprotResponse(res)
	}

	// Create multipart-form
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("file", "list.txt")

	if err != nil {
		return nil, err
	}

	for _, query := range ids {
		if _, err = fw.Write([]byte(query + "\n")); err != nil {
			return nil, err
		}
	}

	if err = w.WriteField("format", "xml"); err != nil {
		return nil, err
	}

	if err = w.WriteField("from", "ACC+ID"); err != nil {
		return nil, err
	}

	if err = w.WriteField("to", "ACC"); err != nil {
		return nil, err
	}

	w.Close()

	// Create and call request
	req, err := http.NewRequest("POST", "http://www.uniprot.org/uploadlists/", &b)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: time.Duration(120) * time.Second}

	res, err = client.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	return processUniprotResponse(res)
}

func processUniprotResponse(res *http.Response) ([]uniprot, error) {
	code := res.StatusCode

	if 200 <= code && code <= 299 {
		var item uniprotBase

		buf, err := ioutil.ReadAll(res.Body)

		if err != nil {
			return nil, err
		}

		err = xml.Unmarshal(buf, &item)

		if err != nil {
			return nil, err
		}

		for i, entry := range item.Entry {
			item.Entry[i].ID = entry.Accession[0]
		}

		if err != nil {
			return nil, err
		}

		return item.Entry, nil
	}

	if 400 <= code && code <= 499 {
		log.Println(res.Status)
		return nil, errHTTPGetClientErr
	}

	if 500 <= code && code <= 599 {
		return nil, errHTTPGetServerErr
	}

	return nil, errHTTPGetUnknownErr
}

func getUniprot(ids []string) ([]uniprot, error) {
	var cached []uniprot
	var missed []string

	for _, id := range ids {
		item, err := uniprotLoadCache(id)

		if err != nil {
			log.Printf("Failed to load UniProt cache for %s: %s", id, err)
			missed = append(missed, id)
		} else {
			cached = append(cached, item)
		}
	}

	if len(missed) > 0 {
		list, err := fetchUniprot(missed)

		if err != nil {
			log.Printf("Failed to fetch from Uniprot: %s", err)
		}

		for _, item := range list {
			if err := item.SaveCache(); err != nil {
				log.Printf("Failed to save UniProt cache for %s: %s", item.ID, err)
			}

			cached = append(cached, item)
		}
	}

	return cached, nil
}
