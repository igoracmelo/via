package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	stations, err := getStations()
	if err != nil {
		panic(err)
	}
	fmt.Println(stations)

	plan, err := getTripPlan("austin", "bangu", "2023-03-10", "15:00")
	if err != nil {
		panic(err)
	}
	fmt.Println(plan)
}

func getStations() (*StationsResponse, error) {
	resp, err := http.Get("https://content.supervia.com.br/estacoes")
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	stations := &StationsResponse{}
	err = json.Unmarshal(body, stations)
	if err != nil {
		return nil, err
	}

	return stations, nil
}

func getTripPlan(from string, to string, date string, stime string) (*TripPlanResponse, error) {
	url := fmt.Sprintf("https://content.supervia.com.br/planeje/%s/%s/%s/%s", from, to, date, stime)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
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

type TripPlanResponse struct {
	Trajetos []struct {
		Viagens [][]struct {
			StationIdOrigin string `json:"estacao_origem_id"`
			StationIdDest   string `json:"estacao_destino_id"`
			TimeDeparture   string `json:"horario_partida"`
			TimeArrival     string `json:"horario_chegada"`
			ExtensionId     string `json:"ramal_id"`
			ExtensionName   string `json:"ramal_nome"`
		}
	}
}

type StationsResponse struct {
	Stations []struct {
		Id   string `json:"id"`
		Name string `json:"nome"`
	} `json:"estacoes"`
}
