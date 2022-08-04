package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pccr10001/mod-xmltv-generator/xmltv"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Channel struct {
	ContentTitle  string
	ChannelNumber int
	ContentId     int
	SdUrl         string
	HdUrl         sql.NullString
	FourKUrl      sql.NullString
}

type Program struct {
	Id           int64
	ContentTitle string
	ContentId    int
	StartTime    time.Time
	EndTime      time.Time
	Description  string
}

func main() {

	db, err := sql.Open("sqlite3", "file:epg_all.sqlite?cache=shared")
	if err != nil {
		log.Println(err)
	}
	db.SetMaxOpenConns(1)
	var mu = `#EXTM3U name="中華電信 MOD"
#EXTREM: Hinet MOD Playlist
#EXTREM:
#EXTREM: Chinese
`
	var epg xmltv.TV
	epg.SourceInfoName = "CHT MOD"
	epg.SourceDataURL = "https://mod.cht.com.tw"
	epg.GeneratorInfoName = "mod-xmltv-generator"
	epg.GeneratorInfoURL = "https://github.com/pccr10001/mod-xmltv-generator"

	r, err := db.Query("SELECT contentTitle, ContentId, channelNumber, sdURL, hdURL, fourKURL from tab_live_channel")
	for r.Next() {
		var c Channel
		_ = r.Scan(&c.ContentTitle, &c.ContentId, &c.ChannelNumber, &c.SdUrl, &c.HdUrl, &c.FourKUrl)
		var u = c.SdUrl
		if c.HdUrl.Valid && c.HdUrl.String != "" {
			u = c.HdUrl.String
		}
		if c.FourKUrl.Valid && c.FourKUrl.String != "" {
			u = c.FourKUrl.String
		}

		// no link
		if u == "" {
			continue
		}

		mu = mu + fmt.Sprintf("#EXTINF:-1 tvg-id=\"%d\" tvg-chno=\"%d\",%s\n%s\n",
			c.ContentId,
			c.ChannelNumber,
			c.ContentTitle,
			strings.Replace(u, "igmp://", "udp://@", -1))
		epg.Channels = append(epg.Channels, xmltv.Channel{
			DisplayNames: []xmltv.CommonElement{{
				Lang:  "zh",
				Value: c.ContentTitle,
			}},
			LCN: c.ChannelNumber,
			ID:  strconv.Itoa(c.ContentId),
		})
	}
	f, _ := os.OpenFile("mod.m3u8", os.O_CREATE|os.O_WRONLY, 0666)
	_, _ = io.WriteString(f, mu)
	_ = f.Close()

	var cstZone = time.FixedZone("CST", 0)

	r, err = db.Query("SELECT programId, contentId, programName, startTime, endTime, description from tab_epg")
	for r.Next() {
		var p Program
		_ = r.Scan(&p.Id, &p.ContentId, &p.ContentTitle, &p.StartTime, &p.EndTime, &p.Description)

		epg.Programmes = append(epg.Programmes, xmltv.Programme{
			Titles: []xmltv.CommonElement{{
				Lang:  "zh",
				Value: p.ContentTitle,
			}},
			Descriptions: []xmltv.CommonElement{{
				Lang:  "zh",
				Value: p.Description,
			}},
			Start:   &xmltv.Time{Time: p.StartTime.In(cstZone)},
			Stop:    &xmltv.Time{Time: p.EndTime.In(cstZone)},
			Channel: strconv.Itoa(p.ContentId),
		})
	}

	f, _ = os.OpenFile("mod_epg.xml", os.O_CREATE|os.O_WRONLY, 0666)
	e, _ := xml.Marshal(epg)
	_, _ = io.WriteString(f, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	_, _ = io.WriteString(f, "<!DOCTYPE tv SYSTEM \"http://api.torrent-tv.ru/xmltv.dtd\">\n")
	_, _ = io.WriteString(f, string(e))
	_ = f.Close()

}
