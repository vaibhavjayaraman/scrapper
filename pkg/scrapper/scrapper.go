package scrapper

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/tidwall/gjson"
	"github.com/worldhistorymap/backend/pkg/shared"
	"go.uber.org/zap"
)

type WikipediaAllPagesQuery struct {
	ApFrom  string `url:"apfrom ,omitempty"`
	Action  string `url:"query , omitempty"`
	List    string `url:"list , omitempty"`
	ApLimit int    `url:"aplimit, omitempty"`
	Format  string `url:"format , omitempty"`
}

type WikipediaGeoSearchQuery struct {
	Action string `url:"query, omitempty"`
	Prop   string `url:"prop, omitempty"`
	Titles string `url:"titles, omitempty"`
	Format string `url:"json", omitempty"`
}

type Article struct {
	Title  string
	PageID string
	Source string
	Lat    float64
	Lon    float64
}

type Scrapper struct {
	DB     *sql.DB
	query  string
	logger *zap.Logger
}

func NewScrapper(config *shared.Config, logger *zap.Logger) error {
	var err error
	query := "INSERT INTO  markers(pageid, title, lat, lon, source, geom) VAlUES($1, '$2', $3, $4, 'wikipedia', ST_SetSRID(ST_MakePoint($3, $4), 4326))"
	connstr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", config.Host, config.Port, config.User,
		config.Password, config.DBName)
	s := Scrapper{}
	s.DB, err = sql.Open("postgres", connstr)
	if err != nil {
		return err
	}
	s.query = query
	if err := s.scrapWikipedia(); err != nil {
		return err
	}
	return nil
}

func isApContEmpty(apcont string) bool {
	return apcont != ""
}

func (s *Scrapper) scrapWikipedia() error {
	var cont bool
	params := &WikipediaAllPagesQuery{
		Format:  "json",
		List:    "allpages",
		Action:  "query",
		ApLimit: 500,
	}

	for cont = true; cont; cont = isApContEmpty(params.ApFrom) {
		client := &http.Client{Timeout: time.Second * 30}
		req, err := http.NewRequest(
			"GET",
			fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&format=json&aplimit=500&list=allpages&apfrom=%s", params.ApFrom),
			nil,
		)
		//req, err := sling.New().Get("https://en.wikipedia.org/w/api.php").QueryStruct(params).Request()
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		pages := gjson.Get(string(body), "query.allpages").Array()
		s.logger.Info(fmt.Sprintf("Got next %d starting from article %s ", len(pages), params.ApFrom))
		for _, page := range pages {
			articleInfo := page.Map()
			pageID := articleInfo["pageid"].Int()
			title := articleInfo["title"].String()
			log.Printf(fmt.Sprintf("Reading in Page %s", title))
			lat, lon, err := getArticleLatLon(pageID, title)
			if err != nil {
				/**log error **/
				s.logger.Error(fmt.Sprintf("Failed to get coords %d", pageID), zap.Error(err))
				continue
			}
			go s.updateDB(pageID, title, lat, lon)
		}
		params.ApFrom = gjson.Get(string(body), "continue.apcontinue").String()
	}
	return nil
}

func (s *Scrapper) updateDB(pageid int64, title string, lat, lon float64) {
	_, err := s.DB.Exec(s.query, pageid, title, lat, lon)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Update DB: %d", pageid))
	}
}

func getArticleLatLon(pageID int64, title string) (float64, float64, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&format=json&titles=%s&prop=coordinates", title),
		nil,
	)
	if err != nil {
		return 0, 0, err
	}

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}
	coordinates := gjson.Get(string(body), fmt.Sprintf("query.pages.%d.coordinates", pageID)).Array()
	if len(coordinates) == 0 {
		return 0, 0, err
	}
	coordinate := coordinates[0].Map()
	lat := coordinate["lat"].Float()
	lon := coordinate["lon"].Float()
	return lat, lon, nil
}
