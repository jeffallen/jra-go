package main

import (
	"flag"
	"log"
	"time"

	"github.com/crackcomm/cloudflare"
	"google.golang.org/cloud/compute/metadata"

	"golang.org/x/net/context"
)

var user = flag.String("user", "", "CloudFlare username")
var key = flag.String("key", "", "CloudFlare API key")

func main() {
	flag.Parse()

	exip, err := metadata.ExternalIP()
	if err != nil {
		log.Fatal(err)
	}

	client := cloudflare.New(&cloudflare.Options{
		Email: *user,
		Key:   *key,
	})

	ctx := context.Background()
	ctx, _ = context.WithDeadline(ctx, time.Now().Add(time.Second*30))

	zones, err := client.Zones.List(ctx)
	if err != nil {
		log.Fatal(err)
	} else if len(zones) == 0 {
		log.Fatal("No zones were found")
	} else if len(zones) != 1 {
		log.Fatal("More than one zone found?")
	}

	if zones[0].Name != "nella.org" {
		log.Fatal("not nella.org?")
	}

	records, err := client.Records.List(ctx, zones[0].ID)
	if err != nil {
		log.Fatal(err)
	}

	// remove all existing ns.nella.org records
	for _, record := range records {
		if record.Name == "ns.nella.org" {
			log.Print("delete ", record.Content)
			err = client.Records.Delete(ctx, record.ZoneID, record.ID)
			if err != nil {
				log.Fatal("delete: ", err)
			}
		}
	}

	rec := &cloudflare.Record{
		Type:    "A",
		Name:    "ns.nella.org",
		Content: exip,
		TTL:     120,
		ZoneID:  zones[0].ID,
	}
	log.Print("create: ", exip)
	err = client.Records.Create(ctx, rec)
	if err != nil {
		log.Fatal("create: ", err)
	}
}
