// bplot is used for visual analysis of cryptocurrencies
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
// ffmpeg -framerate 60 -i ./btcusd_%03d.png -c:v libvpx-vp9 -pix_fmt yuva420p -lossless 1 out.webm

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

// qm is a map of time.Time types to a *quote type
var qm map[time.Time]*quote = make(map[time.Time]*quote)

// days is the nmber of days of data in our CSV file. Some data has multiple
// quotes per day and so make sure you account for that when figuring this
// number up.
var days time.Duration = time.Duration(5242)

// st is the start time. We could start anywhere, but here, we wish to start at
// the begining, which is July 18 2010.
var st time.Time = time.Date(2010, time.July, 18, 0, 0, 0, 0, time.UTC)

// et is the end time of the data contained in the data set. In this case we
// add to the start time of July 18 2010 the number of days specified above in
// hours.
var et time.Time = time.Date(2010, time.July, 18, 0, 0, 0, 0, time.UTC).Add(time.Hour * 24 * (days))

func main() {
	qs := parseQuotes("btc_daily.csv")
	qm = mkQuotes(qs)
	totalPngFrames := days / 10
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

// parseQuotes() takes our data set, a CSV file containing time-series data on
// bitcoin, and splits the string by each new line, returning them a slice,
// each containing unprocessed quote data.
func parseQuotes(quoteFile string) (quotes_ []string) {
	b, err := os.ReadFile(quoteFile)
	if err != nil {
		log.Println(err)
	}
	quotes_ = strings.Split(string(b), "\n")[1:]
	return
}

// mkQuotes() takes the slice of unprocessed quotes returned by parseQuotes()
// and processes it, transforming it into our *quote type data structure,
// creating structured data we can then use for deep analysis, allowing us to
// acquire hidden insights and valuable knowledge that few will ever know. We
// also use a map to map a time.Time to each *quote, creating a handy dandy,
// useful and convenient, nifty little pocket-size go-anywhere do anything
// time-series in a [next word here]. We cycle through the lines, parsing the
// quote data accordingly, and finally, adding it to the map:
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

// newPlot() is a helper function that reduces the code base/typing a little.
// There may be other default options one could add here in the future.
func newPlot() *plot.Plot {
	p := plot.New()
	p.Add(plotter.NewGrid())
	p.HideAxes()

	// Here we format the float to 2 decimal places so that its not
	// atrociously long.
	s := strconv.FormatFloat(qm[et].End, 'f', 2, 64)

	// Style the graph
	p.Title.Text = et.Format(
		"Σ(firma)  |  BTC Deflationary Pattern "+
			"Visualization\n\nJan 02 2006") +
		"      1 BTC = $" + s + " USD"
	p.Title.TextStyle.Font.Size = 25
	p.Title.TextStyle.YAlign = -1.5
	p.Title.TextStyle.Font.Variant = "Mono"
	p.Title.Padding = 50
	// p.Title.TextStyle.XAlign = 0

	return p
}

// mkgr() is used to further design the graph, see what's inside to learn more.
func mkgr(i int) error {
	var (
		// Get the defaults
		p *plot.Plot = newPlot()

		// get the lines
		price  *plotter.Line = getLine(pricePlot(), color.RGBA{R: 192, G: 92, B: 63, A: 1})
		mCap   *plotter.Line = getLine(mcPlot(), color.RGBA{R: 255, G: 99, B: 255, A: 255})
		volume *plotter.Line = getLine(volumePlot(), color.RGBA{R: 255, G: 255, B: 20, A: 255})
	)

	// Add the line(s) to the chart
	p.Add(price)
	p.Add(mCap)
	p.Add(volume)

	// Save a frame
	w, h := mkDimensions(480, 270)
	path := formatFileNameCount(i)
	if err := p.Save(w, h, path); err != nil {
		return err
	}
	return nil
}

// mkDimensions() returns the width (w) and height (h) dimensions that we
// intend our chart to embolden. mkDimensions() reduces the code base/typing by
// creating a type of function I just decided to call a shorthandler, because
// it could be thought of as "short hand" for what it does.
func mkDimensions(w, h float64) (vg.Length, vg.Length) {
	return +vg.Points(w * vg.Millimeter.Points()),
		vg.Points(h * vg.Millimeter.Points())
}

// formatFileNameCount() formats the file name count to keep the files we
// create in order.
func formatFileNameCount(i int) string {
	c := fmt.Sprint(i)
	if len(c) == 1 {
		return fmt.Sprintf("600/FRAME_00%s.png", c)
	}
	return fmt.Sprintf("600/FRAME_0%s.png", c)

}

func getLine(p plotter.XYs, c color.RGBA) (l *plotter.Line) {
	l, err := plotter.NewLine(p)
	if err != nil {
		panic(err)
	}

	l.LineStyle.Width = vg.Points(1)
	l.LineStyle.Color = c
	return
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
