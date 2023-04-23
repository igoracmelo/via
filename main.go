package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/igoracmelo/via/cache"
	"github.com/igoracmelo/via/supervia"
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
	log.SetFlags(0)
	log.SetPrefix("")

	pCmd := flag.NewFlagSet("p", flag.ExitOnError)
	pCmd.StringVar(&planTime, "t", "", "hora da viagem")
	pCmd.StringVar(&planDay, "d", "", "dia da viagem")

	if len(os.Args) < 2 {
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

		if from == "" {
			log.Fatal("estação de origem não identificada: " + os.Args[2])
		}
		if to == "" {
			log.Fatal("estação de destino não identificada: " + os.Args[3])
		}

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

		_, err = supervia.GetAlerts(to, from, planDay, planTime)
		if err != nil {
			log.Panic(err)
		}

		t, err := time.Parse("2006-01-02 15:04", planDay+" "+planTime)
		if err != nil {
			log.Panic(err)
		}

		fmt.Printf("planejando para %s\n\n", humanReadableTime(t))

		plan, err := supervia.GetTripPlan(from, to, planDay, planTime)
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

func printTrajects(plan *supervia.TripPlanResponse) {
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
			arrival := strings.Repeat(" ", 7)
			if len(last.TimeArrival) > 5 {
				arrival = last.TimeArrival[:5] + " - "
			}
			fmt.Println(arrival + color(last.StationNameDest, "bwhite"))

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

func getStations() (*supervia.StationsResponse, error) {
	stations, err := getStationsCache()
	if err == nil {
		return stations, nil
	}

	stations, err = supervia.GetStationsOnline()
	if err != nil {
		return nil, err
	}

	err = storeStationsCache(stations)
	if err != nil {
		log.Print(err)
	}

	return stations, nil
}

func getStationsCache() (*supervia.StationsResponse, error) {
	stations := &supervia.StationsResponse{}

	data, err := cache.Load("stations")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, stations)
	if err != nil {
		return nil, err
	}

	return stations, nil
}

func storeStationsCache(stations *supervia.StationsResponse) error {
	data, err := json.Marshal(stations)
	if err != nil {
		return err
	}

	err = cache.Store("stations", data, 2*24*time.Hour)
	if err != nil {
		return err
	}

	return nil
}

func printRequest(req *http.Request) {
	// fmt.Println(req.Method, req.URL.String())
}

func findStationBestMatch(station string, stations *supervia.StationsResponse) string {
	station = strings.ToLower(station)

	for _, entry := range stations.Stations {
		name := strings.ToLower(entry.Name)
		if strings.Contains(name, station) || strings.Contains(entry.Id, station) {
			return entry.Id
		}
	}

	return ""
}
