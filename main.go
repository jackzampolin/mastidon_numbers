package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	client "github.com/influxdata/influxdb/client/v2"
	"github.com/robfig/cron"
)

type Instances struct {
	Instances []struct {
		ID                string    `json:"_id"`
		AddedAt           time.Time `json:"addedAt,omitempty"`
		Name              string    `json:"name"`
		Downchecks        int       `json:"downchecks"`
		Upchecks          int       `json:"upchecks"`
		HTTPSRank         string    `json:"https_rank,omitempty"`
		HTTPSScore        int       `json:"https_score,omitempty"`
		ObsRank           string    `json:"obs_rank,omitempty"`
		ObsScore          int       `json:"obs_score,omitempty"`
		Ipv6              bool      `json:"ipv6,omitempty"`
		Up                bool      `json:"up"`
		Users             int       `json:"users,omitempty"`
		UsersChangeRatio  int       `json:"usersChangeRatio,omitempty"`
		Statuses          int       `json:"statuses,omitempty"`
		Connections       int       `json:"connections,omitempty"`
		OpenRegistrations bool      `json:"openRegistrations,omitempty"`
		Uptime            float64   `json:"uptime"`
		Infos             struct {
			OptOut                 bool          `json:"optOut"`
			ShortDescription       string        `json:"shortDescription"`
			FullDescription        string        `json:"fullDescription"`
			Theme                  interface{}   `json:"theme"`
			Languages              []string      `json:"languages"`
			NoOtherLanguages       bool          `json:"noOtherLanguages"`
			ProhibitedContent      []string      `json:"prohibitedContent"`
			OtherProhibitedContent []interface{} `json:"otherProhibitedContent"`
			Federation             string        `json:"federation"`
			Bots                   string        `json:"bots"`
			Brands                 string        `json:"brands"`
		} `json:"infos,omitempty"`
		Version         string    `json:"version,omitempty"`
		VersionScore    int       `json:"version_score,omitempty"`
		HistoryMigrated bool      `json:"historyMigrated,omitempty"`
		UpdatedAt       time.Time `json:"updatedAt,omitempty"`
		Second          int       `json:"second"`
		CheckedAt       time.Time `json:"checkedAt"`
		Info            string    `json:"info,omitempty"`
		UptimeStr       string    `json:"uptime_str"`
		Score           float64   `json:"score"`
		Dead            bool      `json:"dead,omitempty"`
		Connected       int       `json:"connected,omitempty"`
		Blacklisted     bool      `json:"blacklisted,omitempty"`
		Date            time.Time `json:"date,omitempty"`
	} `json:"instances"`
	TotalUsers int `json:"totalUsers"`
	Languages  []struct {
		Iso6391       string   `json:"iso639_1"`
		Iso6392       string   `json:"iso639_2"`
		Iso6392En     string   `json:"iso639_2en"`
		Iso6393       string   `json:"iso639_3"`
		Name          []string `json:"name"`
		NativeName    []string `json:"nativeName"`
		Direction     string   `json:"direction"`
		Family        string   `json:"family"`
		Countries     []string `json:"countries,omitempty"`
		LangCultureMs []struct {
			LangCultureName string `json:"langCultureName"`
			DisplayName     string `json:"displayName"`
			CultureCode     string `json:"cultureCode"`
		} `json:"langCultureMs,omitempty"`
	} `json:"languages"`
	Countries []struct {
		Code2         string   `json:"code_2"`
		Code3         string   `json:"code_3"`
		NumCode       string   `json:"numCode"`
		Name          string   `json:"name"`
		Languages     []string `json:"languages,omitempty"`
		LangCultureMs []struct {
			LangCultureName string `json:"langCultureName"`
			DisplayName     string `json:"displayName"`
			CultureCode     string `json:"cultureCode"`
		} `json:"langCultureMs,omitempty"`
	} `json:"countries"`
	ProhibitedContent struct {
		NudityNocw          string `json:"nudity_nocw"`
		NudityAll           string `json:"nudity_all"`
		PornographyNocw     string `json:"pornography_nocw"`
		PornographyAll      string `json:"pornography_all"`
		Sexism              string `json:"sexism"`
		Racism              string `json:"racism"`
		IllegalContentLinks string `json:"illegalContentLinks"`
		Spam                string `json:"spam"`
		Advertising         string `json:"advertising"`
		HateSpeeches        string `json:"hateSpeeches"`
		Harrassment         string `json:"harrassment"`
		SpoilersNocw        string `json:"spoilers_nocw"`
		Array               []struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"array"`
	} `json:"prohibitedContent"`
}

func (i Instances) AboveUsers(count int) int {
	c := 0
	for _, inst := range i.Instances {
		if inst.Users > count {
			c++
		}
	}
	return c
}

func toStr(i int) string {
	return fmt.Sprintf("%v", i)
}

func toStrB(i bool) string {
	return fmt.Sprintf("%v", i)
}

func (i Instances) InstancesPoints() []*client.Point {
	pts := []*client.Point{}
	for _, inst := range i.Instances {
		opc := "no"
		if len(inst.Infos.OtherProhibitedContent) > 0 {
			opc = "yes"
		}
		tags := map[string]string{
			"id":                     inst.ID,
			"name":                   inst.Name,
			"httpsRank":              inst.HTTPSRank,
			"obsRank":                inst.ObsRank,
			"up":                     toStrB(inst.Up),
			"openRegistrations":      toStrB(inst.OpenRegistrations),
			"optOut":                 toStrB(inst.Infos.OptOut),
			"noOtherLanguages":       toStrB(inst.Infos.NoOtherLanguages),
			"federation":             inst.Infos.Federation,
			"bots":                   inst.Infos.Bots,
			"brands":                 inst.Infos.Brands,
			"version":                inst.Version,
			"dead":                   toStrB(inst.Dead),
			"blacklisted":            toStrB(inst.Blacklisted),
			"otherProhibitedContent": opc,
			"nudityNocw":             "allowed",
			"nudityAll":              "allowed",
			"pornographyNocw":        "allowed",
			"pornographyAll":         "allowed",
			"sexism":                 "allowed",
			"racism":                 "allowed",
			"illegalContentLinks":    "allowed",
			"spam":                   "allowed",
			"advertising":            "allowed",
			"hateSpeeches":           "allowed",
			"harrassment":            "allowed",
			"spoilersNocw":           "allowed",
		}
		for _, pro := range inst.Infos.ProhibitedContent {
			if tags[pro] == "allowed" {
				tags[pro] = "banned"
			}
		}
		fields := map[string]interface{}{
			"score":            inst.Score,
			"connected":        inst.Connected,
			"languages":        len(inst.Infos.Languages),
			"obsScore":         inst.ObsScore,
			"httpsScore":       inst.HTTPSScore,
			"usersChangeRatio": inst.UsersChangeRatio,
			"downchecks":       inst.Downchecks,
			"upchecks":         inst.Upchecks,
			"uptime":           inst.Uptime,
			"users":            inst.Users,
			"statuses":         inst.Statuses,
			"connections":      inst.Connections,
			"totalUsers":       i.TotalUsers,
			"totalInstances":   len(i.Instances),
		}
		m := "instances"
		pt, err := client.NewPoint(m, tags, fields, time.Now())
		if err != nil {
			log.Fatal(err)
		}
		pts = append(pts, pt)
	}
	return pts
}

func (i Instances) TotalPoint() *client.Point {
	statuses := 0
	for _, inst := range i.Instances {
		statuses += inst.Statuses
	}
	tags := map[string]string{
		"scrapedFrom": "https://instances.mastodon.xyz/list.json",
	}
	fields := map[string]interface{}{
		"totalUsers":     i.TotalUsers,
		"totalInstances": len(i.Instances),
		"totalStatuses":  statuses,
		"above2":         i.AboveUsers(2),
		"above5":         i.AboveUsers(5),
		"above10":        i.AboveUsers(10),
		"above50":        i.AboveUsers(50),
		"above100":       i.AboveUsers(100),
		"above500":       i.AboveUsers(500),
		"above1000":      i.AboveUsers(1000),
		"above5000":      i.AboveUsers(5000),
		"above10000":     i.AboveUsers(10000),
	}
	m := "totals"
	pt, err := client.NewPoint(m, tags, fields, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	return pt
}

func idb() (client.Client, client.BatchPoints) {
	conn := os.Getenv("INFLUXDB_CONNECTION")
	if conn == "" {
		conn = "http://localhost:8086"
	}
	c, err := client.NewHTTPClient(client.HTTPConfig{Addr: conn})
	if err != nil {
		log.Fatal(err)
	}
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  "mastidon",
		Precision: "s",
	})
	if err != nil {
		log.Fatal(err)
	}
	return c, bp
}

func getData() Instances {
	// Make request to mastodon api to pull data
	resp, err := retryablehttp.Get("https://instances.mastodon.xyz/list.json")
	if err != nil {
		log.Fatal(err)
	}

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// JSON -> Struct
	var r Instances
	json.Unmarshal(body, &r)

	// Return data
	return r
}

func doStuff() {
	// Read CSV file, for now
	r := getData()

	// Instantiate InfluxDB Client
	c, bp := idb()

	// Pull points out of JSON
	pt := r.TotalPoint()
	bp.AddPoint(pt)
	pts := r.InstancesPoints()
	bp.AddPoints(pts)

	// Write data
	err := c.Write(bp)
	if err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func main() {
	doStuff()
	c := cron.New()
	c.AddFunc("@hourly", func() { doStuff() })
	c.Start()
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
