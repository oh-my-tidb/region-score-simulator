package main

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/render"
)

var (
	defaultStores         = []float64{500, 600, 800}
	defaultK      float64 = 1
	defaultM      float64 = 256
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		stores := defaultStores
		if storesStr := r.Form.Get("stores"); storesStr != "" {
			ss := strings.Split(storesStr, "_")
			stores = nil
			for _, s := range ss {
				v, _ := strconv.ParseInt(s, 10, 64)
				stores = append(stores, float64(v))
			}
		}
		var k float64 = defaultK
		if kStr := r.Form.Get("k"); kStr != "" {
			k, _ = strconv.ParseFloat(kStr, 16)
		}
		var m float64 = defaultM
		if mStr := r.Form.Get("m"); mStr != "" {
			m, _ = strconv.ParseFloat(mStr, 16)
		}
		charts := genChart(stores, k, m)
		myRender(w, charts)
	})
	http.ListenAndServe(":8081", nil)
}

func myRender(w http.ResponseWriter, charts []render.Renderer) {
	bufs := make([]bytes.Buffer, len(charts))
	for i, c := range charts {
		c.Render(&bufs[i])
	}
	for i, b := range bufs {
		s := b.String()
		if i == 0 {
			s = strings.Split(s, "<style>")[0]
		} else if i == len(bufs)-1 {
			s = strings.Split(s, "<body>")[1]
		} else {
			s = strings.Split(s, "<body>")[1]
			s = strings.Split(s, "<style>")[0]
		}
		w.Write([]byte(s))
	}
}

func score(R, C, A, K, M float64) float64 {
	if A >= C {
		return R
	}
	return (K + M*(math.Log(C)-math.Log(A))/(C-A)) * R
}

func genChart(Cs []float64, K, M float64) []render.Renderer {
	Rs := make([]float64, len(Cs))
	var xAxis []string
	sizeData := make([][]opts.LineData, len(Cs))
	availableData := make([][]opts.LineData, len(Cs))
	percentData := make([][]opts.LineData, len(Cs))
	for i := 0; ; i++ {
		var minIndex []int
		var minScore float64 = math.MaxFloat64
		for j := range Cs {
			A := Cs[j] - Rs[j]
			if A <= 0 {
				continue
			}
			S := score(Rs[j], Cs[j], A, K, M)
			if S < minScore {
				minIndex, minScore = []int{j}, S
			}
			if S == minScore {
				minIndex = append(minIndex, j)
			}
		}
		if len(minIndex) == 0 {
			break
		}

		Rs[minIndex[rand.Intn(len(minIndex))]] += 0.1
		if i%50 == 0 {
			xAxis = append(xAxis, "")
			for j := range Rs {
				sizeData[j] = append(sizeData[j], opts.LineData{Value: Rs[j]})
				if a := Cs[j] - Rs[j]; a > 0 {
					availableData[j] = append(availableData[j], opts.LineData{Value: a})
				}
				percentData[j] = append(percentData[j], opts.LineData{Value: Rs[j] / Cs[j]})
			}
		}
	}

	size := charts.NewLine()
	size.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Size",
		}),
	)
	size.SetXAxis(xAxis)
	for i := range sizeData {
		size.AddSeries(fmt.Sprintf("s%.0f", Cs[i]), sizeData[i])
	}

	available := charts.NewLine()
	available.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Available",
		}),
	)
	available.SetXAxis(xAxis)
	for i := range availableData {
		available.AddSeries(fmt.Sprintf("s%.0f", Cs[i]), availableData[i])
	}

	percent := charts.NewLine()
	percent.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Percent",
		}),
	)
	percent.SetXAxis(xAxis)
	for i := range percentData {
		percent.AddSeries(fmt.Sprintf("s%.0f", Cs[i]), percentData[i])
	}

	return []render.Renderer{size, available, percent}
}
