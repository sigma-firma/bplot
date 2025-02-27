// bplot is used as a visual analysis tool that processes time-series quote
// data such as that used for trading stocks, bonds, commodities, currencies,
// mutual funds, exchange-traded funds (ETFs), derivatives, cryptocurrencies,
// real estate, etc. This software was originally written in haste by someone
// higher than a kite on LSD. We read data stored in a .csv file into a slice
// of lines, each containing quote data for a single day, then parse each line,
// transforming the csv-structured data into go-structured data, that can then
// be analyzed and used to run tests, simulations, or whatever the fuck else
// you want you awesome mother fucker. This program currently creates a line
// graph showing the price, one for each day (line of the .csv file), which are
// then intended to be stitched into a video using ffmpeg. One day bplot will
// have more features.
//
// To work properly, bplot currently expects a .csv file with the following
// column layout:
//
// 	Start,End,Open,High,Low,Close,Volume,Market Cap
//
// We prefer a video of around ~10 seconds in length and ~60 frames per second.
// This means we need to produce 10*60 == ~600 frames. One frame for each day
// right? Wrong. The number of days of data will change literally every day, so
// we normalize the scale of our time-series to fit within these constraints.
//
// At the time of this comment, we have 5242 days of data (lines) in our .csv
// file. Since we're only creating 600 frames, we want to pick out 600 days,
// equally spaced apart. So we divide 5242 by 600: 5242 / 600 = 8.73666. This
// number contains the devils number, but that's not why it's a problem, it's
// a problem because it's not a whole number. There are a number of different
// ways to handle this situation, and here we do it wrong. We should probably
// round the result to its closest whole number, in our case 9, and then at the
// very end (len(unprocessed) < 9) just take the last day.
//
// We could also make the number of lines/days evenly divisible by 600 by just
// skipping a number of lines from the beginning, I doubt anyone would notice.
// Just don't tell the boss what you did. Blame me if he finds out. If he fires
// me I'm telling him I slept with his wife.
//
// Tip: ffmpeg to make webm:
// ffmpeg -framerate 60 -i ./btcusd_%03d.png -c:v libvpx-vp9 -pix_fmt yuva420p -lossless 1 out.webm
//
// Now onto the real voodoo:

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

// quote is a stock or crypto quote or whatever else takes this form.
type quote struct {
	Date    time.Time // Just guess
	DateEnd time.Time // I dont even know
	Open    float64   // Day Open price
	High    float64   // Day high price
	Low     float64   // Day low price
	End     float64   // Day end price
	// Volume (not sure if this is average high or what).
	Volume float64
	// Market Cap (again not sure).
	MarketCap float64
	// QMap is used in doPlot() to make it so
	// that we can pass an arg to doPlot() that tells it what metric to
	// retrieve. The quote struct its self maybe unnecessary. This software
	// was originally written in haste and higher than a kite on LSD.
	QMap map[string]float64
}

var quotes_path string = "btc_daily.csv"

// qm is a map of keys of type time.Time to values of type *quote.
var qm map[time.Time]*quote = make(map[time.Time]*quote)

// _24h is a shorthander for time.Hour*24 (1 day in hours). Instantiated with
// the hopes of making some things a little easier to grok.
var _24h time.Duration = time.Hour * 24

// days is the number of days of data in our CSV file. Some data has multiple
// quotes per day and so make sure you account for that when figuring this
// number up.
var days time.Duration = time.Duration(5242)

// st is the start time. We could start anywhere, but here, we wish to start at
// the beginning, which in our case is July 18 2010.
var st time.Time = time.Date(2010, time.July, 18, 0, 0, 0, 0, time.UTC)

// et is the end time of the data contained in the data set. In this case we
// add to the start time of July 18 2010 the number of days specified above in
// hours.
var et time.Time = st.Add(time.Hour * 24 * (days))

// totalFrames is used to determine the total number of individual frames we
// wish to create. Here, we crudely divide the number of days (5242 atm), by
// 10, which equals 524.2 frames. The external/third party tool "ffmpeg" will
// be used to stitch the frames together into a video, thus, the actual FPS
// will be determined by that command. The more images we create here (and the
// larger their dimensions), the longer this program will take to run. It is
// advised that low quality settings be used for testing, and quality settings
// used for the finalized product.
var totalFrames time.Duration = days / 10 // close enough for day-scale data.

// plots is used as a mechanism in the hopes of creating a cleaner code base.
// When we make a frame (graph/chart/wtfever), we sometimes want to plot
// multiple lines on the same graph, but they sometimes won't fit, so we
// "scale" them to fit within the confines of our parameters. With a stock
// quote you may want to overlay the volume and market cap. The number the keys
// are mapped to will be divided into what the actual key value amounts to in
// &quote.QMap[key] = value
// NOTE: This implementation is absolute garbage atm. It will be fixed but I
// don't know if I want everyone to know how I do it.
var plots map[string]float64 = map[string]float64{
	"End": 1, "MarketCap": 2000000, "Volume": 2000000,
}

// red is used to color the price line.
var red *color.RGBA = &color.RGBA{R: 192, G: 92, B: 63, A: 1}

// var blu *color.RGBA = &color.RGBA{R: 255, G: 99, B: 255, A: 255}
// var yel *color.RGBA = &color.RGBA{R: 255, G: 255, B: 20, A: 255}

func main() {
	// Structure the quotes.
	mkQuotes(parseQuotes(quotes_path))

	// For each frame...
	for i := 0; i <= int(totalFrames); i++ {
		// Make and save a single frame.
		doFrame(i, mkLine(doPlot("End"), red))

		incrTime(time.Duration(i)) // see: updateEndTime()

		// A stylish progress bar so you're not left wondering.
		progressOutput(i, totalFrames)
	}
}

// incrTime(): This voodoo here determines how many hours to skip between
// frames, we take the days and divide the quantity by the number of frames,
// times the number of frames we've already processed:
// (5,000 / 500) * 50 = 500 hours
func incrTime(framesDone time.Duration) {
	et = st.Add(_24h * (days / totalFrames * framesDone))
}

// doFrame() is used to make the graphs, saving them to the appropriate path as,
// well. See what's inside to learn more.
func doFrame(i int, l *plotter.Line) {
	// Add line with defaults to the chart.
	var p *plot.Plot = newPlot()
	p.Add(l)

	// Chart width and height in pixels. The bigger, the longer the program
	// will take to run, and the longer ffmpeg will take to run. You have
	// been warned but no one can stop you. DO IT.
	var w, h vg.Length = mkDimensions(480, 270)

	// Build & save this frame, returning as the value any error.
	if p.Save(w, h, formatFileNameCount(i)) != nil {
		log.Println("Error saving.")
	}
}

// doPlot() is used to plot the price variable, returning a plotter.XYs
// type which implements the plotter interface, and is basically the line its
// self, just not on a chart. The line is placed atop the chart in the next
// episode of the twilight zone.
//
//	func doPlot(data string, count int) plotter.XYs {
//		// Our main loop is constantly modifying et
//		pts := make(plotter.XYs, count)
//		for i := 0; i < count; i++ {
//			pts[i].Y = qm[st.Add(time.Duration(days-i)*_24h*days)].QMap[data]
//			log.Println(pts[i].Y)
//		}
//		return pts
//	}
//
// func pricePlot() plotter.XYs {
func doPlot(data string) plotter.XYs {
	pts := make(plotter.XYs, et.Sub(st)/(time.Hour*24))
	for i := range pts {
		pts[i].X = float64(i)
		if pts[i].X >= float64(et.Sub(st)/(time.Hour*24)) {
			break
		}
		pts[i].Y = qm[st.Add(time.Hour*24*time.Duration(i))].QMap[data]
	}
	return pts
}

// mkDimensions() returns the width (w) and height (h) dimensions that we
// intend our chart to embolden. mkDimensions() reduces the code base/typing by
// creating a type of function I just decided to call a shorthandler, because
// it could be thought of as "short hand" for what it does.
func mkDimensions(w, h float64) (vg.Length, vg.Length) {
	return vg.Points(w * vg.Millimeter.Points()),
		vg.Points(h * vg.Millimeter.Points())
}

// formatFileNameCount() formats the file name count to keep the files we
// create in order. This is necessary because my file system has decided that
// 10.png comes before 2.png, unless the "numbers" are reformatted with equal
// numbers of characters by prepending one or two zeros like so:
//
// 1.png  -->  001.png
// 2.png  -->  002.png
//
// When we get to 10.png we must add one zero:
//
// 10.png  -->  010.png
//
// If we wanted more than 999 frames we'd have to add another if statement, or
// ideally re-write this function so that it can detect how many zeros you need
// to add onto any number.
func formatFileNameCount(i int) string {
	if len(fmt.Sprint(i)) == 1 {
		return fmt.Sprintf("600/FRAME_00%d.png", i)
	}
	if len(fmt.Sprint(i)) == 2 {
		return fmt.Sprintf("600/FRAME_0%d.png", i)

	}
	return fmt.Sprintf("600/FRAME_%d.png", i)
}

// mkLine() returns a *plotter.Line with some sane defaults.
func mkLine(p plotter.XYs, c *color.RGBA) (l *plotter.Line) {
	l, err := plotter.NewLine(p)
	if err != nil {
		panic(err)
	}

	l.LineStyle.Width = vg.Points(1)
	l.LineStyle.Color = c
	return
}

// Boring stuff below here. You have been warned!

// progressOutput() is used to display a useful progress bar in the terminal
// while the frames are being processed.
func progressOutput(i int, totalPngFrames time.Duration) {
	// clear the terminal (this is what makes it look animated).
	clearTerm()

	// This is a little heading to let us know everything is okay.
	fmt.Printf("\nCreating %d frames from %d days of market data:\n\n",
		int(totalPngFrames), int(days))

	// This voodoo here determines the progress in terms of percent. It
	// looks ridiculous to me, but I wrote it and usually I never write
	// incorrect code but this just doesn't look right. I must've been
	// high.
	percentDone := int(math.Floor(((1.00/(float64(totalPngFrames)/
		(float64(i)+0.01)))*100.00)*100) / 100)

	// No idea why this is needed.
	if percentDone >= 100 {
		percentDone = 100
	}

	// loading will be a progress bar, displayed to the user as terminal
	// output. blanks is used for formatting and provides padding for
	// the progress bar so the text doesn't shift. People notice bad
	// formatting!
	loading, blanks := "", ""

	// Make the loading bar using percentDone.
	for q := 0; q <= int(percentDone)/6; q++ {
		loading = loading + "="
	}

	// Make the padding by adding spaces to the blanks var.
	for y := 0; y <= (100/6)-len(loading); y++ {
		if len(loading) <= 100/6 {
			blanks = blanks + " "
		} else {
			blanks = ""
		}
	}

	// Finally, we print a neatly formatted string, indicating the progress
	// with a nifty progress bar, and all it took was the will to make it
	// happen.
	fmt.Printf(" [%s %s %d%%]  --  frame %d of %d\n\n",
		loading, blanks, percentDone, i, int(totalPngFrames))
}

// clearTerm() is used to clear the terminal when updating the progress bar
// displaid by progressBarOutput(). To do this it runs fmt.Println() 200 times,
// but this number is excessive and could be reduced. Also the function could
// take an arg to determine this number.
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
func mkQuotes(quotes_ []string) {
	for _, q := range quotes_ {
		if len(q) > 2 {
			quote_ := strings.Split(q, ",")
			newQuote := &quote{QMap: make(map[string]float64)}
			date, err := time.Parse(time.DateOnly,
				strings.TrimSpace(quote_[0]))
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
			newQuote.QMap["End"] = price

			newQuote.MarketCap = marketCap
			newQuote.QMap["MarketCap"] = marketCap

			newQuote.Volume = volume
			newQuote.QMap["Volume"] = volume

			qm[date] = newQuote
		}
	}
}

// newPlot() is a helper function that reduces the code base/typing a little.
// There may be other default options one could add here in the future.
func newPlot() *plot.Plot {
	p := plot.New()
	p.Add(plotter.NewGrid())
	p.HideAxes()

	// Here we format the float to 2 decimal places so that its not
	// atrociously long.
	// s := strconv.FormatFloat(qm[et].End, 'f', 2, 64)

	// Style the graph
	// p.Title.Text = et.Format(title_txt) + s + " USD"
	p.Title.Padding = 50
	p.Title.TextStyle.Font.Variant = "Mono"
	p.Title.TextStyle.Font.Size = 25
	p.Title.TextStyle.YAlign = -1.5
	// p.Title.TextStyle.XAlign = 0

	return p
}

var title_txt string = "Σ(firma)  |  BTC Deflationary Pattern Visualization" +
	"\n\nJan 02 2006      1 BTC = $"
