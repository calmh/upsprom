package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	listenAddr := flag.String("listen", "localhost:2112", "Address to listen on")
	upsdAddr := flag.String("upsd", "localhost:3493", "Address of upsd")
	flag.Parse()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(*listenAddr, nil))
	}()

	for {
		if err := process(*upsdAddr); err != nil {
			log.Printf("Error: %v", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func process(upsdAddr string) error {
	conn, err := net.Dial("tcp", upsdAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	br := bufio.NewScanner(conn)

	var ups []string
	fmt.Fprintf(conn, "LIST UPS\n")
upsList:
	for br.Scan() {
		line := br.Text()
		fields := strings.Fields(line)
		switch fields[0] {
		case "UPS":
			ups = append(ups, fields[1])
		case "END":
			break upsList
		}
	}
	if br.Err() != nil {
		return br.Err()
	}

	vars := make(map[string]*prometheus.GaugeVec)

	for {
	varList:
		for _, u := range ups {
			fmt.Fprintf(conn, "LIST VAR %s\n", u)
			for br.Scan() {
				line := br.Text()
				fields := strings.Fields(line)
				switch fields[0] {
				case "VAR":
					key := strings.ReplaceAll(fields[2], ".", "_")
					val := strings.Trim(fields[3], "\"")
					if fval, err := strconv.ParseFloat(val, 64); err == nil {
						if _, ok := vars[key]; !ok {
							vars[key] = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: key}, []string{"ups"})
							prometheus.MustRegister(vars[key])
						}
						vars[key].WithLabelValues(u).Set(fval)
					}
				case "END":
					break varList
				}
			}
			if br.Err() != nil {
				return br.Err()
			}
		}
		time.Sleep(5 * time.Second)
	}
}
