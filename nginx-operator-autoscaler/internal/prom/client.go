package prom

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type resp struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// InstantVector runs an instant query and returns a single float64 sum.
func InstantVector(promURL, query string) (float64, error) {
	u, _ := url.Parse(promURL)
	u.Path = "/api/v1/query"
	q := u.Query()
	q.Set("query", query)
	u.RawQuery = q.Encode()

	r, err := http.Get(u.String())
	if err != nil {
		return 0, err
	}
	defer r.Body.Close()

	var out resp
	if err := json.NewDecoder(r.Body).Decode(&out); err != nil {
		return 0, err
	}
	if out.Status != "success" || len(out.Data.Result) == 0 || len(out.Data.Result[0].Value) < 2 {
		return 0, nil
	}
	// value[1] is string numeric
	s, ok := out.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, fmt.Errorf("unexpected result format")
	}
	var f float64
	_, err = fmt.Sscan(s, &f)
	return f, err
}
