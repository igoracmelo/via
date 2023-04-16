package supervia

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type StationsResponse struct {
	Stations []struct {
		Id   string `json:"id"`
		Name string `json:"nome"`
	} `json:"estacoes"`
}

func GetStationsOnline() (*StationsResponse, error) {
	req, err := http.NewRequest("GET", "https://content.supervia.com.br/estacoes", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d\nbody: %s", resp.StatusCode, string(body))
	}

	if err != nil {
		return nil, err
	}

	stations := &StationsResponse{}
	err = json.Unmarshal(body, stations)
	return stations, err
}

type TripPlanResponse struct {
	Trajects []struct {
		Trips [][]struct {
			StationIdOrigin   string `json:"estacao_origem_id"`
			StationNameOrigin string `json:"estacao_origem_nome"`
			StationIdDest     string `json:"estacao_destino_id"`
			StationNameDest   string `json:"estacao_destino_nome"`
			TimeDeparture     string `json:"horario_partida"`
			TimeArrival       string `json:"horario_chegada"`
			ExtensionId       string `json:"ramal_id"`
			ExtensionName     string `json:"ramal_nome"`
		} `json:"viagens"`
	} `json:"trajetos"`
}

func GetTripPlan(from string, to string, sdate string, stime string) (*TripPlanResponse, error) {
	url := fmt.Sprintf("https://content.supervia.com.br/planeje/%s/%s/%s/%s", from, to, sdate, stime)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d\nbody: %s", resp.StatusCode, string(body))
	}
	if err != nil {
		return nil, err
	}

	plan := &TripPlanResponse{}
	err = json.Unmarshal(body, plan)
	if err != nil {
		return nil, err
	}

	return plan, nil
}

// TODO:
type AlertsResponse []struct{}

func GetAlerts(to, from, sdate, stime string) (*AlertsResponse, error) {
	q := url.Values{}

	fields := []string{
		"nid",
		"title",
		"field_alerta_ramais",
		"field_alerta_estacao",
		"field_alerta_descricao",
		"field_alerta_data",
		"field_alerta_link",
	}

	q.Set("type", "alerta")
	q.Set("fields", strings.Join(fields, ","))
	q.Set("partida", to)
	q.Set("chegada", from)
	q.Set("data", sdate)
	q.Set("hora", stime)
	// q.Add("ramais", ...)

	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme:   "https",
			Host:     "www.supervia.com.br",
			Path:     "/pt-br/api/alertas",
			RawQuery: q.Encode(),
		},
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d\nbody: %s", resp.StatusCode, string(body))
	}
	if err != nil {
		return nil, err
	}

	alerts := &AlertsResponse{}
	err = json.Unmarshal(body, alerts)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}
