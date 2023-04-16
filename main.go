package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
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

var planDay string
var planTime string

func main() {
	pCmd := flag.NewFlagSet("p", flag.ExitOnError)
	pCmd.StringVar(&planTime, "t", "", "hora da viagem")
	pCmd.StringVar(&planDay, "d", "", "dia da viagem")

	if len(os.Args) < 1 {
		log.Panic("argumentos insuficientes")
	}

	switch os.Args[1] {
	case "p":
		if len(os.Args) < 4 {
			log.Panic("argumentos insuficientes")
		}

		if len(os.Args) > 4 {
			err := pCmd.Parse(os.Args[4:])
			if err != nil {
				log.Panic(err)
			}
		}

		stations, err := getStations()
		if err != nil {
			log.Panic(err)
		}

		from := findStationBestMatch(os.Args[2], stations)
		to := findStationBestMatch(os.Args[3], stations)

		now := time.Now().Add(2 * time.Minute)

		if planDay == "" {
			planDay = now.Format("2006-01-02")
		} else {
			planDay = formatPlanDay(planDay, now)
		}

		if planTime == "" {
			planTime = now.Format("15:04")
		} else {
			planTime = formatPlanTime(planTime, now)
		}

		_, err = getAlerts(to, from, planDay, planTime)
		if err != nil {
			log.Panic(err)
		}

		t, err := time.Parse("2006-01-02 15:04", planDay+" "+planTime)
		if err != nil {
			log.Panic(err)
		}

		fmt.Printf("planejando para %s\n\n", humanReadableTime(t))

		plan, err := getTripPlan(from, to, planDay, planTime)
		if err != nil {
			log.Panic(err)
		}

		printTrajects(plan)
	}
}

func humanReadableTime(t time.Time) string {
	_, offset := time.Now().Zone()
	now := time.Now().UTC().Add(time.Duration(offset) * time.Second)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	tomorrow := today.Add(24 * time.Hour)
	afterTomorrow := tomorrow.Add(24 * time.Hour)
	in30min := now.Add(30 * time.Minute)

	s := ""

	if t.After(today) && t.Before(tomorrow) {
		s += "hoje "
	} else if t.After(tomorrow) && t.Before(afterTomorrow) {
		s += "amanhã "
	} else if t.Month() == now.Month() && t.Year() == now.Year() {
		s += "dia " + t.Format("02") + " "
	} else if t.Year() == now.Year() {
		s += "dia " + t.Format("02/01") + " "
	} else {
		s += "dia " + t.Format("02/01/06") + " "
	}

	if t.After(now) && t.Before(in30min) {
		mins := int(math.Floor(t.Sub(now).Minutes()))
		if mins == 0 {
			s += "agora"
		} else if mins == 1 {
			s += "em " + fmt.Sprint(mins) + " minuto"
		} else {
			s += "em " + fmt.Sprint(mins) + " minutos"
		}
	} else {
		s += "as " + t.Format("15:04") + " horas"
	}

	return s
}

func printTrajects(plan *TripPlanResponse) {
	for i, traject := range plan.Trajects {
		fmt.Println(color("Trajeto", "bwhite"), i+1)
		fmt.Println()

		for j, trip := range traject.Trips {
			fmt.Println("> Opção", j+1)

			for _, subtrip := range trip {
				dep := strings.Repeat(" ", 7)
				if len(subtrip.TimeDeparture) > 5 {
					dep = subtrip.TimeDeparture[:5] + " - "
				}
				fmt.Println(dep + color(subtrip.StationNameOrigin, "bwhite"))
				fmt.Println(color("          |", subtrip.ExtensionId))
			}

			last := trip[len(trip)-1]
			fmt.Printf("%s - %s\n", last.TimeArrival[:5], color(last.StationNameDest, "bwhite"))

			fmt.Println()
			fmt.Println()
		}
	}
}

func formatPlanTime(s string, fallback time.Time) string {
	if regexp.MustCompile(`^\d{2}:\d{2}`).MatchString(s) {
		return s
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback.Format("15:04")
	}

	if n < 10 {
		s = "0" + s
	}

	s += ":00"
	return s
}

func formatPlanDay(s string, fallback time.Time) string {
	splits := strings.Split(s, "/")
	d := splits[0]

	m := ""
	if len(splits) > 1 {
		m = splits[1]
	} else {
		m = fallback.Format("01")
	}

	y := ""
	if len(splits) > 2 {
		y = splits[2]
	} else {
		y = fallback.Format("2006")
	}

	return fmt.Sprintf("%s-%s-%s", y, m, d)
}

func getStations() (*StationsResponse, error) {
	stations, err := getStationsCache()
	if err == nil {
		return stations, nil
	}

	stations, err = getStationsOnline()
	if err != nil {
		return nil, err
	}

	err = storeStationsCache(stations)
	if err != nil {
		log.Print(err)
	}

	return stations, nil
}

func getStationsOnline() (*StationsResponse, error) {
	req, err := http.NewRequest("GET", "https://content.supervia.com.br/estacoes", nil)
	if err != nil {
		return nil, err
	}

	printRequest(req)
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

func getStationsCache() (*StationsResponse, error) {
	p := path.Join(os.TempDir(), "via-stations-cache")
	stations := &StationsResponse{}

	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, stations)
	if err != nil {
		return nil, err
	}

	return stations, nil
}

func storeStationsCache(stations *StationsResponse) error {
	p := path.Join(os.TempDir(), "via-stations-cache")

	data, err := json.Marshal(stations)
	if err != nil {
		return err
	}

	err = os.WriteFile(p, data, 0666)
	if err != nil {
		return err
	}

	return nil
}

func getTripPlan(from string, to string, sdate string, stime string) (*TripPlanResponse, error) {
	url := fmt.Sprintf("https://content.supervia.com.br/planeje/%s/%s/%s/%s", from, to, sdate, stime)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	printRequest(req)
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

func getAlerts(to, from, sdate, stime string) (*AlertsResponse, error) {
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

	printRequest(req)
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

func printRequest(req *http.Request) {
	// fmt.Println(req.Method, req.URL.String())
}

func findStationBestMatch(station string, stations *StationsResponse) string {
	station = strings.ToLower(station)
	bestId := ""
	bestText := ""

	for _, entry := range stations.Stations {
		bestDelta := len(bestId) - len(bestText)
		currDelta := len(entry.Id) - len(station)

		if strings.Contains(entry.Id, station) && (bestId == "" || currDelta < bestDelta) {
			bestId = entry.Id
			bestText = station
		}
	}

	return bestId
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

type AlertsResponse []struct{}
