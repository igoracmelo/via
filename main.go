package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var colors = map[string]string{
	"santa_cruz": "\033[0;32m",
	"paracambi":  "\033[0;36m",
	"japeri":     "\033[0;34m",
	"saracuruna": "\033[0;33m",
	"bwhite":     "\033[1m",
}

func color(s string, name string) string {
	return colors[name] + s + "\033[0m"
}

func main() {
	if len(os.Args) < 2 {
		panic("argumentos insuficientes") // TODO:
	}

	if os.Args[1] == "p" {
		var (
			from  string
			to    string
			sdate string
			stime string
		)

		if len(os.Args) < 4 {
			panic("argumentos insuficientes") // TODO:
		}

		from = os.Args[2]
		to = os.Args[3]

		if len(os.Args) == 4 {
			now := time.Now().Add(2 * time.Minute)
			sdate = now.Format("2006-01-02")
			stime = now.Format("15:04")

			fmt.Printf("Planejando para daqui há 2 minutos (%s -> %s)\n\n", from, to)
		}

		if len(os.Args) == 5 {
			stime = os.Args[4]
			now := time.Now().Add(2 * time.Minute)
			sdate = now.Format("2006-01-02")

			fmt.Printf("Planejando para hoje as %s (%s -> %s)\n\n", stime, from, to)
		}

		if len(os.Args) == 6 {
			stime = os.Args[4]
			splits := strings.Split(os.Args[5], "/")
			d := splits[0]

			m := ""
			if len(splits) > 1 {
				m = splits[1]
			} else {
				m = time.Now().Format("01")
			}

			y := ""
			if len(splits) > 2 {
				y = splits[2]
			} else {
				y = time.Now().Format("2006")
			}

			sdate = fmt.Sprintf("%s-%s-%s", y, m, d)
			fmt.Printf("Planejando para dia %s/%s/%s as %s (%s -> %s)\n\n", d, m, y, stime, from, to)
		}

		plan, err := getTripPlan(from, to, sdate, stime)
		if err != nil {
			panic(err) // TODO:
		}

		for i, traject := range plan.Trajects {
			fmt.Println(color("Trajeto", "bwhite"), i+1)
			fmt.Println()

			for j, trip := range traject.Trips {
				fmt.Println("> Opção", j+1)

				for _, subtrip := range trip {
					fmt.Printf("%s - %s\n", subtrip.TimeDeparture[:5], color(subtrip.StationNameOrigin, "bwhite"))
					fmt.Println(color("          |", subtrip.ExtensionId))
				}

				last := trip[len(trip)-1]
				fmt.Printf("%s - %s\n", last.TimeArrival[:5], color(last.StationNameDest, "bwhite"))

				fmt.Println()
				fmt.Println()
			}
		}
	}
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

func getTripPlan(from string, to string, sdate string, stime string) (*TripPlanResponse, error) {
	url := fmt.Sprintf("https://content.supervia.com.br/planeje/%s/%s/%s/%s", from, to, sdate, stime)
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

type StationsResponse struct {
	Stations []struct {
		Id   string `json:"id"`
		Name string `json:"nome"`
	} `json:"estacoes"`
}
