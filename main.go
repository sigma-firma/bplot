package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// ffmpeg to make webm:
// fmpeg -framerate 60 -i ./btcusd_%03d.png -c:v libvpx-vp9 -pix_fmt yuva420p -lossless 1 out.webm

type quote struct {
	Date      time.Time
	DateEnd   time.Time
	Open      float64
	High      float64
	Low       float64
	End       float64
	Volume    float64
	MarketCap float64
}

var qm map[time.Time]*quote = make(map[time.Time]*quote)

// var et time.Time = time.Date(2023, time.May, 14, 0, 0, 0, 0, time.UTC)
var days time.Duration = time.Duration(5242)
var st time.Time = time.Date(2010, time.July, 18, 0, 0, 0, 0, time.UTC)
var et time.Time = time.Date(2010, time.July, 18, 0, 0, 0, 0, time.UTC).Add(time.Hour * 24 * (days))

func main() {
	qs := parseQuotes("btc_daily.csv")
	qm = mkQuotes(qs)
	totalPngFrames := days / 50
	for i := 0; i <= int(totalPngFrames); i++ {
		et = time.Date(2010, time.July, 18, 0, 0, 0, 0, time.UTC).Add(time.Hour * 24 * (days / time.Duration(totalPngFrames)) * time.Duration(i))
		mkgr(i)
		clearTerm()
		fmt.Println("Creating", int(totalPngFrames), "frames from", int(days), "days of market data:")
		fmt.Println()
		percentDone := int(math.Floor(((1.00/(float64(totalPngFrames)/(float64(i)+0.01)))*100.00)*100) / 100)
		if percentDone >= 100 {
			percentDone = 100
		}
		loading := ""
		endblanks := ""
		for q := 0; q <= int(percentDone)/6; q++ {
			loading = loading + "="
		}
		for y := 0; y <= (100/6)-len(loading); y++ {
			if len(loading) <= 100/6 {
				endblanks = endblanks + " "
			} else {
				endblanks = ""
			}
		}
		fmt.Println(" [", loading, endblanks, percentDone, "%", "]", "  --   frame", i, "of", int(totalPngFrames))
		fmt.Println()
	}
	fmt.Println("saved in: ./600/*.png")
	fmt.Println()
}

func clearTerm() {
	for i := 0; i <= 200; i++ {
		fmt.Println()
	}
}

func parseQuotes(quoteFile string) (quotes_ []string) {
	b, err := os.ReadFile(quoteFile)
	if err != nil {
		log.Println(err)
	}
	quotes_ = strings.Split(string(b), "\n")[1:]
	fmt.Println(len(quotes_))
	return
}
func mkQuotes(quotes_ []string) map[time.Time]*quote {
	for _, q := range quotes_ {
		if len(q) > 2 {
			quote_ := strings.Split(q, ",")
			newQuote := &quote{}
			date, err := time.Parse(time.DateOnly, strings.TrimSpace(quote_[0]))
			if err != nil {
				log.Println(err)
			}
			price, err := strconv.ParseFloat(quote_[2], 64)
			if err != nil {
				log.Println(err)
			}
			marketCap, err := strconv.ParseFloat(quote_[7], 64)
			if err != nil {
				log.Println(err)
			}
			volume, err := strconv.ParseFloat(quote_[6], 64)
			if err != nil {
				log.Println(err)
			}

			newQuote.Date = date
			newQuote.End = price
			newQuote.MarketCap = marketCap
			newQuote.Volume = volume

			qm[date] = newQuote
		}
	}
	return qm
}

func newPlot() *plot.Plot {
	p := plot.New()
	p.Add(plotter.NewGrid())
	p.HideAxes()
	return p
}

func mkgr(i int) {
	p := newPlot()
	// s := strconv.FormatFloat(qm[et].End, 'f', 2, 64)

	// p.Title.Text = et.String() + " -- " + s
	l, err := plotter.NewLine(pricePlot())
	if err != nil {
		panic(err)
	}
	l.LineStyle.Width = vg.Points(1)
	l.LineStyle.Color = color.RGBA{R: 255, G: 99, B: 71, A: 255}

	mcl, err := plotter.NewLine(mcPlot())
	if err != nil {
		panic(err)
	}
	mcl.LineStyle.Width = vg.Points(1)
	mcl.LineStyle.Color = color.RGBA{R: 255, G: 99, B: 255, A: 255}

	vl, err := plotter.NewLine(volumePlot())
	if err != nil {
		panic(err)
	}
	vl.LineStyle.Width = vg.Points(1)
	vl.LineStyle.Color = color.RGBA{R: 255, G: 255, B: 20, A: 255}

	p.Add(l)
	p.Add(mcl)
	p.Add(vl)
	count := fmt.Sprint(i)
	if len(count) == 1 {
		count = "00" + count
	}
	if len(count) == 2 {
		count = "0" + count
	}

	if err := p.Save(480*vg.Millimeter, vg.Points(270*vg.Millimeter.Points()), "600/btcusd_"+count+".png"); err != nil {
		panic(err)
	}
}

func pricePlot() plotter.XYs {
	pts := make(plotter.XYs, et.Sub(st)/(time.Hour*24))
	for i := range pts {
		pts[i].X = float64(i)
		if pts[i].X >= float64(et.Sub(st)/(time.Hour*24)) {
			break
		}
		pts[i].Y = qm[st.Add(time.Hour*24*time.Duration(i))].End
	}
	return pts
}

func mcPlot() plotter.XYs {
	pts := make(plotter.XYs, et.Sub(st)/(time.Hour*24))
	for i := range pts {
		pts[i].X = float64(i)
		if pts[i].X >= float64(et.Sub(st)/(time.Hour*24)) {
			break
		}
		pts[i].Y = qm[st.Add(time.Hour*24*time.Duration(i))].MarketCap / (20000000)
		// pts[i].Y = qm[st.Add(time.Hour*24*time.Duration(i))].MarketCap / (20000000)
	}
	return pts
}

func volumePlot() plotter.XYs {
	pts := make(plotter.XYs, et.Sub(st)/(time.Hour*24))
	for i := range pts {
		pts[i].X = float64(i)
		if pts[i].X >= float64(et.Sub(st)/(time.Hour*24)) {
			break
		}
		pts[i].Y = qm[st.Add(time.Hour*24*time.Duration(i))].Volume / (2000000)
	}
	return pts
}
