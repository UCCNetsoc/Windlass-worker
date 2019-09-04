package models

import (
	"encoding/json"
	"net/http"
	"time"
)

type APIResponse struct {
	Status  int
	Content interface{}
}

type apiResponse struct {
	Content interface{} `json:"content"`
	Time    time.Time   `json:"time"`
}

func (resp APIResponse) MarshalJSON() ([]byte, error) {
	timed := apiResponse{
		Content: resp.Content,
		Time:    time.Now(),
	}
	return json.Marshal(timed)
}

func (resp APIResponse) Render(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(resp.Status)
	return nil
}
