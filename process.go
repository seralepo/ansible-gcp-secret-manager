package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

const defaultCredsPath = "/tmp/.ansible/gcp_vault_secret_creds.json"

// Input params provided by ansible
type ModuleArgs struct {
	Name                        string `json:"name"`
	CredentialsFile             string `json:"creds_file"`
	ProjectID                   string `json:"project_id"`
	UsePrivateGoogleAPIEndpoint bool   `json:"private_google_api_endpoint"`
}

// Response object, printed to stdout
type Response struct {
	Msg     string `json:"msg"`
	Data    string `json:"data"`
	Changed bool   `json:"changed"`
	Failed  bool   `json:"failed"`
}

func (r *Response) WithError(msg string) *Response {
	r.Failed = true
	r.Msg = msg
	return r
}

func (r *Response) WithErrorf(format string, args ...interface{}) *Response {
	msg := fmt.Sprintf(format, args...)
	return r.WithError(msg)
}

// Main process, invoked by main()
// Returns exit code of the program
func process(ctx context.Context) int {
	var rc int = 0

	// Get response
	response := produceResponse(ctx)

	// Check response status
	if response.Failed {
		rc = 1
	}

	// Marshal response into JSON string
	text, err := json.Marshal(response)
	if err != nil {
		text, _ = json.Marshal(Response{Msg: "Internal error: invalid response object", Failed: true})
		rc = 1
	}

	// Print response to stdout
	fmt.Println(string(text))

	// Return exit code to main()
	return rc
}

// Do the job and return response
func produceResponse(ctx context.Context) *Response {
	response := new(Response)

	if len(os.Args) < 2 {
		return response.WithError("No argument file provided")
	}

	// Read JSON file
	argsFile := os.Args[1]
	text, err := ioutil.ReadFile(argsFile)
	if err != nil {
		return response.WithErrorf("Could not read parameters file %s: %s", argsFile, err.Error())
	}

	// Parse JSON file contents
	var moduleArgs ModuleArgs
	err = json.Unmarshal(text, &moduleArgs)
	if err != nil {
		return response.WithErrorf("Parameters file %s is invalid: %s", argsFile, err.Error())
	}

	// Fail if secret 'name' not provided
	if moduleArgs.Name == "" {
		return response.WithError(`Parameter 'name' is mandatory`)
	}

	// Use default credentials file path, if 'creds_file' is not provided
	if moduleArgs.CredentialsFile == "" {
		moduleArgs.CredentialsFile = defaultCredsPath
	}

	if moduleArgs.CredentialsFile == "system" {
		moduleArgs.CredentialsFile = "system"
	}

	if moduleArgs.CredentialsFile == "system" && moduleArgs.ProjectID == "" {
		return response.WithErrorf("Project ID is required when system is used for credentials.")
	}

	// If 'project_id' is empty, try using one from credentials file
	if moduleArgs.ProjectID == "" {
		// Read credentials file
		credsBinary, err := ioutil.ReadFile(moduleArgs.CredentialsFile)
		if err != nil {
			return response.WithErrorf("Could not read credentials file %s: %s", moduleArgs.CredentialsFile, err.Error())
		}

		// Parse credentials file to retrieve "project_id"
		var creds struct {
			ProjectID string `json:"project_id"`
		}
		err = json.Unmarshal(credsBinary, &creds)
		if err != nil {
			return response.WithErrorf("Credentials file %s is not valid JSON", moduleArgs.CredentialsFile)
		}

		if creds.ProjectID == "" {
			return response.WithError("Parameter 'project_id' is not specified and is missing in credentials file")
		}

		moduleArgs.ProjectID = creds.ProjectID

	}

	// Set path to credentials file
	if moduleArgs.CredentialsFile != "system" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", moduleArgs.CredentialsFile)
	}

	// Create Google Secret Manager Client
	client, err := NewGCPVaultClient(ctx, moduleArgs.ProjectID, moduleArgs.UsePrivateGoogleAPIEndpoint)
	if err != nil {
		return response.WithErrorf("Could not initialize Google API client: %s", err.Error())
	}
	defer client.Close()

	// Retrieve secret data
	data, err := client.GetSecret(ctx, moduleArgs.Name)
	if err != nil {
		return response.WithErrorf("Could not retrieve secret data: %s", err.Error())
	}

	// Fill in fields in successful response
	response.Msg = "Success"
	response.Failed = false
	response.Data = string(data)

	// Return reponse
	return response
}
