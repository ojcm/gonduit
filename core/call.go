package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/karlseguin/typed"
)

// GetEndpointURI formats a hostname and method name into an endpoint URI.
func GetEndpointURI(host string, method string) string {
	return fmt.Sprintf("%s/api/%s", strings.TrimSuffix(host, "/"), method)
}

// Global client to improve connection caching (per godoc).
var client *http.Client

// PerformCall performs a call to the Conduit API with the provided URL and
// parameters. The response will be unmarshaled into the passed result struct.
//
// If an error is encountered, it will be unmarshalled into a ConduitError
// struct.
func PerformCall(
	endpointURL string,
	params interface{},
	result interface{},
	options *ClientOptions,
) error {
	req, err := MakeRequest(endpointURL, params, options)
	if err != nil {
		return err
	}

	if client == nil {
		client = makeHTTPClient(options)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	jsonBody, err := typed.Json(body)
	if err != nil {
		return err
	}

	// parse any error conduit returned first
	if jsonBody.Exists("error_code") && jsonBody.String("error_code") != "" {
		return &ConduitError{
			code: jsonBody.String("error_code"),
			info: jsonBody.String("error_info"),
		}
	}

	if jsonBody.Exists("result") == false {
		return ErrMissingResults
	}

	// If we get no errors, parse the expected result
	resultBytes, err := jsonBody.ToBytes("result")
	if err != nil {
		return err
	}

	if result != nil && resultBytes != nil {
		var arrResult []interface{}

		if err = json.Unmarshal(resultBytes, &arrResult); err == nil {
			if len(arrResult) < 1 {
				return nil
			}
		}

		if err = json.Unmarshal(resultBytes, &result); err != nil {
			return err
		}
	}

	return nil
}
