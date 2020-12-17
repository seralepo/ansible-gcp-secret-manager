package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

const defaultCredsPath = "/tmp/.ansible/gcp_vault_secret_creds.json"

type ModuleArgs struct {
	Name                        string `json:"name"`
	CredentialsFile             string `json:"creds_file"`
	ProjectID                   string `json:"project_id"`
	UsePrivateGoogleAPIEndpoint bool   `json:"private_google_api_endpoint"`
}

type Response struct {
	Msg     string `json:"msg"`
	Data    string `json:"data"`
	Changed bool   `json:"changed"`
	Failed  bool   `json:"failed"`
}

func fail(response *Response) *Response {
	response.Failed = true
	return response
}

func process(ctx context.Context) *Response {
	response := new(Response)

	if len(os.Args) < 2 {
		response.Msg = "No argument file provided"
		return fail(response)
	}

	// Read JSON file
	argsFile := os.Args[1]

	text, err := ioutil.ReadFile(argsFile)
	if err != nil {
		response.Msg = fmt.Sprintf("Could not read configuration file %s: %s", argsFile, err.Error())
		return fail(response)
	}

	// Parse JSON file contents
	var moduleArgs ModuleArgs
	err = json.Unmarshal(text, &moduleArgs)
	if err != nil {
		response.Msg = "Configuration file not valid JSON: " + argsFile
		return fail(response)
	}

	// Fail if secret 'name' not provided
	if moduleArgs.Name == "" {
		response.Msg = `Parameter 'name' is mandatory`
		return fail(response)
	}

	// Use default credentials file path, if 'creds_file' is not provided
	if moduleArgs.CredentialsFile == "" {
		moduleArgs.CredentialsFile = defaultCredsPath
	}

	// If 'project_id' is empty, try using one from credentials file
	if moduleArgs.ProjectID == "" {
		// Read credentials file
		credsBinary, err := ioutil.ReadFile(moduleArgs.CredentialsFile)
		if err != nil {
			response.Msg = fmt.Sprintf("Could not read credentials file %s: %s", moduleArgs.CredentialsFile, err.Error())
			return fail(response)
		}

		// Parse credentials file to retrieve "project_id"
		var creds struct {
			ProjectID string `json:"project_id"`
		}
		err = json.Unmarshal(credsBinary, &creds)
		if err != nil {
			response.Msg = fmt.Sprintf("Credentials file %s is not valid JSON", moduleArgs.CredentialsFile)
			return fail(response)
		}

		moduleArgs.ProjectID = creds.ProjectID
	}

	// Set path to credentials file
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", moduleArgs.CredentialsFile)

	// Create Google Secret Manager Client
	client, err := NewGCPVaultClient(ctx, moduleArgs.ProjectID, moduleArgs.UsePrivateGoogleAPIEndpoint)
	if err != nil {
		response.Msg = fmt.Sprintf("Could not initialize Google API client: %s", err.Error())
		return fail(response)
	}

	defer client.Close()

	// Retrieve secret data
	data, err := client.GetSecret(ctx, moduleArgs.Name)
	if err != nil {
		response.Msg = fmt.Sprintf("Could not retrieve secret data: %s", err.Error())
		return fail(response)
	}

	// Fill in fields in successful response
	response.Msg = "Success"
	response.Failed = false
	response.Data = string(data)

	// Return reponse
	return response
}
