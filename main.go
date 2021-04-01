package main

import (
	"bytes"
	"flag"
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
	defaultStores             = []float64{500, 600, 800}
	defaultSizeAmps           = []float64{1, 1, 1}
	defaultDeadSpaces         = []float64{0, 0, 0}
	defaultK          float64 = 1
	defaultM          float64 = 256
	defaultF          float64 = 20
)

var port = flag.String("p", ":8081", "serving addr")

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		stores := defaultStores
		if storesStr := r.Form.Get("stores"); storesStr != "" {
			stores = parseFloats(storesStr)
		}
		amps := defaultSizeAmps
		if ampsStr := r.Form.Get("amps"); ampsStr != "" {
			amps = parseFloats(ampsStr)
		}
		deadSpaces := defaultDeadSpaces
		if deadSpaceStr := r.Form.Get("deads"); deadSpaceStr != "" {
			deadSpaces = parseFloats(deadSpaceStr)
		}
		var k float64 = defaultK
		if kStr := r.Form.Get("k"); kStr != "" {
			k, _ = strconv.ParseFloat(kStr, 16)
		}
		var m float64 = defaultM
		if mStr := r.Form.Get("m"); mStr != "" {
			m, _ = strconv.ParseFloat(mStr, 16)
		}
		var f float64 = defaultF
		if fStr := r.Form.Get("f"); fStr != "" {
			f, _ = strconv.ParseFloat(fStr, 16)
		}
		charts := genChart(stores, amps, deadSpaces, k, m, f)
		myRender(w, charts)
	})

	fmt.Printf("server start,listen: http://localhost%s", *port)
	http.ListenAndServe(*port, nil)
}

func parseFloats(s string) []float64 {
	var res []float64
	ss := strings.Split(s, "_")
	for _, s := range ss {
		v, _ := strconv.ParseFloat(s, 64)
		res = append(res, v)
	}
	return res
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

func score(R, C, A, K, M, F float64) float64 {
	if A >= C {
		return R
	}
	if A > F {
		return (K + M*(math.Log(C)-math.Log(A-F+1))/(C-A+F-1)) * R
	}
	return (K+M*math.Log(C)/(C))*R + (F-A)*(K+M*math.Log(F)/F)
}

func genChart(Cs, Amps, Ds []float64, K, M, F float64) []render.Renderer {
	Rs := make([]float64, len(Cs))
	var xAxis []float64
	sizeData := make([][]opts.LineData, len(Cs))
	availableData := make([][]opts.LineData, len(Cs))
	percentData := make([][]opts.LineData, len(Cs))
	scoreData := make([][]opts.LineData, len(Cs))
	for i := 0; ; i++ {
		var minIndex []int
		var minScore float64 = math.MaxFloat64
		for j := range Cs {
			A := Cs[j] - Ds[j] - Rs[j]*Amps[j]
			if A <= 0 {
				continue
			}
			S := score(Rs[j], Cs[j], A, K, M, F)
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
			xAxis = append(xAxis, Rs[0])
			for j := range Rs {
				sizeData[j] = append(sizeData[j], opts.LineData{Value: Rs[j]})
				A := Cs[j] - Ds[j] - Rs[j]*Amps[j]
				if A > 0 {
					availableData[j] = append(availableData[j], opts.LineData{Value: A})
				}
				percentData[j] = append(percentData[j], opts.LineData{Value: (Rs[j]*Amps[j] + Ds[j]) / Cs[j]})
				scoreData[j] = append(scoreData[j], opts.LineData{Value: score(Rs[j], Cs[j], A, K, M, F)})
			}
		}
	}

	legends := make([]string, len(Cs))
	for i := range Cs {
		legends[i] = fmt.Sprintf("s-%d", i)
	}
	size := charts.NewLine()
	size.SetGlobalOptions(setDefaultOption(legends)...)

	size.SetXAxis(xAxis)
	for i := range sizeData {
		size.AddSeries(fmt.Sprintf("s-%d", i), sizeData[i])
	}

	available := charts.NewLine()
	available.SetGlobalOptions(setDefaultOption(legends)...)
	available.SetXAxis(xAxis)
	for i := range availableData {
		available.AddSeries(fmt.Sprintf("s-%d", i), availableData[i])
	}

	percent := charts.NewLine()
	percent.SetGlobalOptions(setDefaultOption(legends)...)
	percent.SetXAxis(xAxis)
	for i := range percentData {
		percent.AddSeries(fmt.Sprintf("s-%d", i), percentData[i])
	}

	score := charts.NewLine()
	score.SetGlobalOptions(setDefaultOption(legends)...)
	score.SetXAxis(xAxis)
	for i := range scoreData {
		score.AddSeries(fmt.Sprintf("s-%d", i), scoreData[i])
	}

	return []render.Renderer{size, available, percent, score}
}

func setDefaultOption(legends []string) []charts.GlobalOpts {
	return []charts.GlobalOpts{
		charts.WithTitleOpts(opts.Title{
			Title: "Score",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    true,
			Trigger: "axis",
		}),
		charts.WithLegendOpts(opts.Legend{
			Data: legends,
			Show: true,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Show: true,
			Name: "used size",
		}, 0),
		charts.WithToolboxOpts(opts.Toolbox{
			Show:  true,
			Right: "20%",
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show: true,
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show: true,
				},
			},
		}),
	}
}
