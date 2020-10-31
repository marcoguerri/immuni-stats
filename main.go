package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
	"net/http"

	en "github.com/marcoguerri/immuni-stats/exposure-notification-interface"
	"time"
)

// Uses exposure notfication format based on
// https://developers.google.com/android/exposure-notifications/exposure-key-file-format

// Format of metadata from "https://get.immuni.gov.it/v1/keys/index"
type meta struct {
	Oldest int
	Newest int
}

func fetch(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("could not fetch url %d: %+v", url, err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("could not fetch url %s, status code: %d, %s", url, resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("could not ready url body (%s): %+v", url, err)
	}
	return body
}

func getTEKs(data []byte) (*en.TemporaryExposureKeyExport, error) {
	hdr := []byte("EK Export v1\x20\x20\x20\x20")
	if bytes.Compare(data[:16], hdr) != 0 {
		return nil, fmt.Errorf("header not recognized: %x", data[:16])
	}

	TEKs := &en.TemporaryExposureKeyExport{}
	if err := proto.Unmarshal(data[16:], TEKs); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal TEK: %w", err)
	}
	return TEKs, nil
}

func main() {

	// Fetch metadata:
	metaBody := fetch("https://get.immuni.gov.it/v1/keys/index")
	metadata := meta{}
	if err := json.Unmarshal(metaBody, &metadata); err != nil {
		log.Fatalf("could not unmarshal metadata (`%s`): %+v", metaBody, err)
	}

	log.Printf("meta: oldest %d, newest %d", metadata.Oldest, metadata.Newest)
	if metadata.Oldest > metadata.Newest {
		log.Fatalf("meta oldest cannot be > than newest")
	}

	total := uint64(0)

	var firstTimestamp, lastTimestamp time.Time
	for keys := metadata.Oldest; keys <= metadata.Newest; keys++ {

		url := fmt.Sprintf("https://get.immuni.gov.it/v1/keys/%d", keys)
		log.Printf("fetching %s", url)
		keysBody := fetch(url)
		reader, err := zip.NewReader(bytes.NewReader(keysBody), int64(len(keysBody)))
		if err != nil {
			log.Fatalf("could not read keys body from %s: %+v", url, err)
		}

		for _, file := range reader.File {
			if file.Name == "export.bin" {

				f, err := file.Open()
				if err != nil {
					log.Fatalf("could not open export.bin for %s: %+v", url, err)
				}
				defer f.Close()
				data, err := ioutil.ReadAll(f)
				if err != nil {
					log.Fatalf("could not read content from export.bin for %s: %+v", url, err)
				}

				TEKs, err := getTEKs(data)
				for _, k := range TEKs.Keys {
					if *k.RollingPeriod != int32(144) {
						log.Printf("!! key with rolling period != 144 !!")
					}

				}
				if err != nil {
					log.Fatalf("could not get TEKs from data for url %s: %+v", url, err)
				}
				log.Printf("")
				log.Printf("======== BEGIN Key Batch %4d ========", keys)
				begin := time.Unix(int64(*TEKs.StartTimestamp), 0)
				end := time.Unix(int64(*TEKs.EndTimestamp), 0)

				log.Printf("batch start timestamp: %s", begin)
				log.Printf("batch end timestamp: %s", end)
				log.Printf("time window size: %.1f hours", end.Sub(begin).Hours())
				log.Printf("number of keys: %d", len(TEKs.Keys))
				log.Printf("======== END Key Batch %4d ========", keys)
				total += uint64(len(TEKs.Keys))
				if keys == metadata.Oldest {
					firstTimestamp = begin
				} else if keys == metadata.Newest {
					lastTimestamp = end
				}
				log.Printf("")
			}

		}
	}

	log.Printf("total number of keys: %d", total)
	log.Printf("unique number of reports assuming 14 TEK per report: %d", total/14)
	log.Printf("total time window: %dd %dh", int(lastTimestamp.Sub(firstTimestamp).Hours()/24), int(lastTimestamp.Sub(firstTimestamp).Hours())%24)

}
