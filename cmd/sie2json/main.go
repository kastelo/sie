package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"kastelo.dev/sie/v2"
)

func main() {
	for _, path := range os.Args[1:] {
		f, err := os.Open(path)
		if err != nil {
			log.Fatalf("opening %s: %v", path, err)
		}
		doc, err := sie.Parse(f)
		_ = f.Close()
		if err != nil {
			log.Fatalf("parsing %s: %v", path, err)
		}

		out, err := protojson.Marshal(doc)
		if err != nil {
			log.Fatalf("marshalling %s: %v", path, err)
		}
		var buf bytes.Buffer
		_ = json.Indent(&buf, out, "", "  ")
		out = buf.Bytes()

		outPath := strings.TrimSuffix(path, ".se") + ".json"
		if err := os.WriteFile(outPath, out, 0o644); err != nil {
			log.Fatalf("writing %s: %v", outPath, err)
		}
		log.Printf("%s -> %s", path, outPath)
	}
}
