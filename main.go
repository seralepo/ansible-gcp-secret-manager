package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type ModuleArgs struct {
	// name of the secret
	Name                        string `json:"name"`

	// GCP project ID
	ProjectID                   string `json:"project_id"`

	// Path to Google credentials JSON on remote host
	CredentialsFile             string `json:"creds_file"`

	// Route all requests via private.googleapis.com:443
	// Don't use if not sure 
	UsePrivateGoogleAPIEndpoint bool   `json:"private_google_api_endpoint"`
}

type Response struct {
	Msg     string `json:"msg"`
	Data    string `json:"data"`
	Changed bool   `json:"changed"`
	Failed  bool   `json:"failed"`
}

func ExitJson(responseBody Response) {
	returnResponse(responseBody)
}

func FailJson(responseBody Response) {
	responseBody.Failed = true
	returnResponse(responseBody)
}

// Prints JSON response to stdout and exits
func returnResponse(responseBody Response) {
	var response []byte
	var err error
	response, err = json.Marshal(responseBody)
	if err != nil {
		response, _ = json.Marshal(Response{Msg: "Invalid response object"})
	}
	fmt.Println(string(response))
	if responseBody.Failed {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func main() {
	var response Response

	if len(os.Args) < 2 {
		response.Msg = "No argument file provided"
		FailJson(response)
	}

	// Read JSON file
	argsFile := os.Args[1]

	text, err := ioutil.ReadFile(argsFile)
	if err != nil {
		response.Msg = fmt.Sprintf("Could not read configuration file %s: %s", argsFile, err.Error())
		FailJson(response)
	}

	// Parse JSON file contents
	var moduleArgs ModuleArgs
	err = json.Unmarshal(text, &moduleArgs)
	if err != nil {
		response.Msg = "Configuration file not valid JSON: " + argsFile
		FailJson(response)
	}

	// Perform mandatory args check
	if moduleArgs.Name == "" {
		response.Msg = `parameter 'name' is mandatory`
		FailJson(response)
	}
	if moduleArgs.ProjectID == "" {
		response.Msg = `parameter 'project_id' is mandatory`
		FailJson(response)
	}
	if moduleArgs.CredentialsFile == "" {
		response.Msg = `parameter 'creds_file' is mandatory`
		FailJson(response)
	}

	// Set path to credentials file
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", moduleArgs.CredentialsFile)

	// Create Google Secret Manager Client
	client, err := NewGCPVaultClient(context.TODO(), moduleArgs.ProjectID, moduleArgs.UsePrivateGoogleAPIEndpoint)
	if err != nil {
		response.Msg = fmt.Sprintf("Could not initialize Google API client: %s", err.Error())
		FailJson(response)
	}

	// Retrieve secret data
	dataBinary, err := client.GetSecret(context.TODO(), moduleArgs.Name)
	if err != nil {
		response.Msg = fmt.Sprintf("Could not retrieve secret data: %s", err.Error())
		FailJson(response)
	}

	// Fill in fields in successful response
	response.Data = string(dataBinary)
	response.Msg = "Success"

	// Return reponse
	ExitJson(response)
}
