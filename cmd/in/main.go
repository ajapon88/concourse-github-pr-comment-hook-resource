package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ajapon88/concourse-github-pr-comment-hook-resource"
)

type Request struct {
	Source  resource.Source  `json:"source"`
	Version resource.Version `json:"version"`
	Params  Params           `json:"params"`
}

type Response struct {
	Version  resource.Version        `json:"version"`
	Metadata []resource.MetadataPair `json:"metadata"`
}

type Params struct {
}

func main() {
	var request Request
	decoder := json.NewDecoder(os.Stdin)
	err := decoder.Decode(&request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to decode: %s\n", err.Error())
		os.Exit(1)
		return
	}

	dest := os.Args[1]
	fmt.Fprintf(os.Stderr, "dest: %s\n", dest)
	fmt.Fprintf(os.Stderr, "version: %v\n", request.Version)
	fmt.Fprintf(os.Stderr, "source: %v\n", request.Source)
	fmt.Fprintf(os.Stderr, "params: %v\n", request.Params)

	response := Response{
		resource.Version{},
		[]resource.MetadataPair{},
	}
	json.NewEncoder(os.Stdout).Encode(response)
}
