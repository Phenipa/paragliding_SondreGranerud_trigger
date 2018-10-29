package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

const databaseName = "paragliding_igc"
const collectionName = "tracks"
const webhookCollection = "webhooks"

type webhook struct {
	URL            string        `json:"webhookURL"`
	TriggerValue   int64         `json:"minTriggerValue"`
	TriggerCounter int64         `json:"triggercounter"`
	PreviousSeenID string        `json:"previoustrigger" bson:"previoustrigger"`
	ID             bson.ObjectId `json:"id" bson:"_id"`
}

type jsonTrack struct { //Helper-struct to appropriately respon      d with data about a requested track.
	Pilot       string        `json:"pilot"`
	Hdate       string        `json:"h_date"`
	Glider      string        `json:"glider"`
	GliderID    string        `json:"glider_id"`
	TrackLength float64       `json:"track_length"`
	URL         string        `json:"url"`
	ID          bson.ObjectId `json:"id" bson:"_id"`
}

func main() {
	starttime := time.Now()
	session, err := mgo.Dial(os.Getenv("DBURL"))
	if err != nil {
		log.Fatal("Database-connection could not be made: ", err)
		return
	}
	defer session.Close()
	cweb := session.DB(databaseName).C(webhookCollection)
	ctra := session.DB(databaseName).C(collectionName)
	amounthooks, err := cweb.Count()
	if err != nil {
		log.Fatal("Could not get amount of webhooks: ", err)
		return
	}
	if amounthooks == 0 {
		fmt.Println(amounthooks)
		return
	}
	amounttracks, err := ctra.Count()
	if err != nil {
		log.Fatal("Could not get amount of tracks: ", err)
		return
	}
	webhooks := make([]webhook, amounthooks, amounthooks)
	err = cweb.Find(nil).All(&webhooks)
	if err != nil {
		log.Fatal("Could not find all webhooks: ", err)
	}
	for _, w := range webhooks {
		var result []*jsonTrack
		var oneres jsonTrack
		if bson.IsObjectIdHex(w.PreviousSeenID) {
			ctra.Find(bson.M{"_id": bson.M{"$gt": bson.ObjectIdHex(w.PreviousSeenID)}}).All(&result)
			w.TriggerCounter += int64(len(result))
			if w.TriggerCounter >= w.TriggerValue {
				fmt.Println(w)
				ids := make([]string, len(result))
				for i, tracks := range result {
					ids[i] = tracks.ID.Hex()
				}
				latest := result[len(result)-1].ID.Time().Unix()
				format := struct {
					Text string `json:"text"`
				}{
					fmt.Sprintf("The latest track has timestamp: %d, the last added tracks are: %v. This took %fseconds to process.",
						latest,
						ids,
						time.Since(starttime).Seconds()),
				}
				fmt.Println(fmt.Sprintf("%+v", format))
				body := new(bytes.Buffer)
				_ = json.NewEncoder(body).Encode(format)
				req, _ := http.NewRequest("POST", w.URL, body)
				req.Header.Set("Content-Type", "application/json")
				fmt.Println(req)
				client := &http.Client{}
				resp, err := client.Do(req)
				fmt.Println(resp)
				if resp.Body != nil {
					defer resp.Body.Close()
				}
				if err != nil {
					log.Fatal("Could not post webhook: ", err)
				}
				w.TriggerCounter = 0
				w.PreviousSeenID = result[len(result)-1].ID.Hex()
				cweb.UpsertId(w.ID, bson.M{"$set": bson.M{"previoustrigger": result[len(result)-1].ID.Hex()}})
			}
		} else {
			if err = ctra.Find(nil).Skip(amounttracks - 1).One(&oneres); err != nil {
				log.Fatal("Could not find track: ", err)
			}
			cweb.UpsertId(w.ID, bson.M{"$set": bson.M{"previoustrigger": oneres.ID.Hex()}})
		}
	}
}
