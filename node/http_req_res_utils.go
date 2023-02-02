package node

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

func writeErrorResponse(w http.ResponseWriter, err error) {
	errorJson, _ := json.Marshal(ErrorResponse{err.Error()})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_, err = w.Write(errorJson)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func writeResponse(w http.ResponseWriter, content interface{}) {
	contentJson, err := json.Marshal(content)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(contentJson)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func readRequest(r *http.Request, requestBody interface{}) error {
	requestBodyJson, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read request body. %s", err.Error())
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
	}(r.Body)

	err = json.Unmarshal(requestBodyJson, requestBody)
	if err != nil {
		return fmt.Errorf("unable to unmarshal request body. %s", err.Error())
	}

	return nil
}

func readResponse(r *http.Response, responseBody interface{}) error {
	responseBodyJson, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read response body. %s", err.Error())
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
	}(r.Body)

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to process response. %s", string(responseBodyJson))
	}

	err = json.Unmarshal(responseBodyJson, responseBody)
	if err != nil {
		return fmt.Errorf("unable to unmarshal response body. %s", err.Error())
	}

	return nil
}
