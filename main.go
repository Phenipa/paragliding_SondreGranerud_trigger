package main

import (
	"github.com/globalsign/mgo"
	"os"
	"log"
)

const databaseName := "paragliding"
const collectionName := "tracks"
const webhookCollection := "webhooks"

type clock struct {
	PreviousCheckTime int64
	
}

func main() {
	session, err mgo.Dial(os.Getenv("DBURL"))
	if err != nil {
		log.Fatal("Database-connection could not be made", err)
		return
	}
	defer session.Close()
	c := session.DB(databaseName).C(webhookCollection)
	amount, err := c.Count()
}
