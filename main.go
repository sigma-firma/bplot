// bplot is used as a visual analysis tool for processing time-series quote
// data such as that used for trading stocks, bonds, commodities, currencies,
// mutual funds, exchange-traded funds (ETFs), derivatives, cryptocurrencies,
// real estate, etc.
//
// We read data stored in .csv format into a slice of lines, each containing
// quote data for a single day, then parse each line, transforming the
// csv-structured data into go-structured data, that can then be analyzed and
// used to run tests, simulations, or whatever the fuck else you want you
// awesome mother fucker.
//
// The bplot program, in its current state, creates a line graph plotting the
// price of the trade-able asset, one for each day (line of the .csv file),
// which are then intended to be stitched into a 10-second video using ffmpeg.
// One day bplot will have more features.
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
// number contains the devils number, but that's not it's problem, it's problem
// is that it's not a whole number. There are a number of different ways to
// handle this situation, and here, we do it wrong. We should probably round
// the result to its closest whole number, in our case 9, and then at the very
// end (len(unprocessed) < 9) just take the last day.
//
// We could also make the number of lines/days evenly divisible by 600 by just
// skipping a number of lines from the beginning, I doubt anyone would notice.
// Just don't tell the boss what you did.
//
// Tip: ffmpeg to make webm:
// ffmpeg -framerate 60 -i ./btcusd_%03d.png -c:v libvpx-vp9 -pix_fmt yuva420p -lossless 1 out.webm
//
// Now onto the *real* voodoo:

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

// quotes_path is the path bplot looks for your .csv file containing your
// quote data. If you don't have this or don't know what the hell I'm talking
// about, this is equivocal to showing up on test day with no pants.
var quotes_path string = "btc_daily.csv"

// qm is a map of keys of type time.Time to values of type *quote. This is
// basically a pocket-sized time-series few will even take notice of, but it's
// right here, hiding in plain site, just waiting to burst out onto the scene,
// and change the world for better or for worse or for what the fuck ever.
var qm map[time.Time]map[string]float64 = make(map[time.Time]map[string]float64)

// title_txt is used by our company, who ever the fuck can edit this code, or
// whoever tf pays us to edit it, in the formatting of the title text that will
// be displayed obnoxiously across our entire video. You want this or the chart
// will look like its made by someone who has no idea wtf they're doing. We
// know, and we're here to make $10,000,000 minimum. Fuck all non-believers.
var title_txt string = "Σ(firma)  |  BTC Deflationary Pattern Visualization" +
	"\n\nJan 02 2006      1 BTC = $"

// day is a shorthander for time.Hour*24 (1 day in hours). Instantiated with
// the hopes of making the voodoo contained herein a comprehensible one.
var day time.Duration = time.Hour * 24

// days is the number of days of data in our CSV file. Some data may have
// multiple quotes per day split up over multiple lines, so make sure you
// account for that when figuring this number up you awesome mother fucker.
var days time.Duration = time.Duration(5242)

// totalFrames is used to determine the total number of individual frames we
// wish to create. Here, we crudely divide the number of days (5242 atm), by
// 10, which equals 524.2 frames, and is close enough to 600 for this type of
// data analysis, cause thhis ain't no fuckin rockit surgery, sergio. The
// external/third party tool "ffmpeg" will be used to stitch the frames
// together into a .webm, thus, the actual FPS will be determined by that
// command. The more images we create here (and the larger the image
// dimensions), the longer bplot and ffmpeg will take to process the
// time-series data. It is advised that low quality settings be used for
// testing, and high quality settings preserved only for the finest of
// finalized products.
var totalFrames time.Duration = days / 10 // close enough for day-scale data.

// st is the start time. We could start anywhere, but here, we wish to start at
// the beginning, which in our case is: July 18 2010.
var st time.Time = time.Date(2010, time.July, 18, 0, 0, 0, 0, time.UTC)

// et is the end time of each graph. For this analysis, which is called a
// fractal analyzer, et starts by graphing the first day of data, then the
// first and second, then first, second and third, etc. Here, we add to the
// start time (st) of July 18 2010 the number of days specified above in hours.
var et time.Time = st.Add(time.Hour * 24 * (days))

// plots is used as a convenience mechanism with real hopes of creating a
// cleaner code base. When a frame is generated by bplot, (graph/chart/w/ever),
// its often desirable to plot multiple lines on the same chart, but usually
// they won't fit, so here, we "scale" them, to fit within the confines placed
// by the parameters holding precedence. With a stock quote for example, it may
// be desirable to overlay the price with volume and market cap indicators.
// The number the keys are mapped to will be divided into what the actual key
// value amounts to in quote.QMap[key] = value
// NOTE: This implementation is absolute garbage. It will be fixed but I
// don't know if I want everyone to know this part of the voodoo. Still though,
// its GPL so fuck it do whatever the fuck you want you INCREDIBLE PROGRAMMER!
var plots map[string]float64 = map[string]float64{
	"End": 1, "MarketCap": 2000000, "Volume": 2000000,
}

// red is used to color the price line.
var red *color.RGBA = &color.RGBA{R: 255, G: 65, B: 36, A: 1}

// var blu *color.RGBA = &color.RGBA{R: 255, G: 99, B: 255, A: 255}
// var yel *color.RGBA = &color.RGBA{R: 255, G: 255, B: 20, A: 255}

func main() {
	// Structure the quotes.
	mkQuotes(parseQuotes(quotes_path))

	// For each frame...
	for i := 0; i <= int(totalFrames); i++ {
		// Increment the end time (et)
		incrTime(time.Duration(i)) // see: updateEndTime()

		// Make and save a single frame.
		doFrame(i, mkLine(doPlot("End"), red))

		// A stylish progress bar so you're not left wondering wtf is
		// going on and how tf long is this fucking thing gonna take.
		progressOutput(i, totalFrames)
	}
}

// incrTime(): This voodoo determines how many hours to skip between frames, we
// take the number of days and divide it by the total number of frames times
// the number of frames we've already processed: (5,000 / 500) * 50 = 500 hours
func incrTime(framesDone time.Duration) {
	et = st.Add(day * (days / totalFrames * framesDone))
}

// doFrame() is used to make the graphs, each one acting as a frame for our
// outoput video, saving them to the appropriate path as well.
func doFrame(i int, l *plotter.Line) {
	// Add line with defaults values to the chart.
	var p *plot.Plot = newPlot()
	p.Add(l)

	// Chart width and height in pixels. The bigger the chart, the longer
	// bplot will take to run, and the longer ffmpeg will take to run. You
	// have been warned... but no one can stop you. DO IT.
	var w, h vg.Length = mkDimensions(1920, 1080)

	// Build & save this frame, returning as the value any error.
	if p.Save(w, h, formatFileNameCount(i)) != nil {
		log.Println("Error saving.")
	}
}

// doPlot() is used to plot the line using the format y = mx + b, and returning
// a plotter.XYs type, implementing the plotter interface. It's literally the
// coke line its self, just not on a chart. The line is placed atop the chart
// in the next function, probably.
func doPlot(data string) plotter.XYs {
	// don't even try to figure this out on LSD
	pts := make(plotter.XYs, et.Sub(st)/day)
	for i := range pts {
		if pts[i].X = float64(i); pts[i].X < float64(len(pts)) {
			pts[i].Y = qm[st.Add(day*time.Duration(i))][data] / plots[data]
		}
	}
	return pts
}

// mkDimensions() returns the width (w) and height (h) dimensions that we
// intend our chart to possess. mkDimensions() reduces the code base/typing by
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
//	1.png  -->  001.png       When we get to
//	2.png  -->  002.png       10.png, we add
//	------------------------> another  zero:    10.png  -->  010.png
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

// mkLine() returns a *plotter.Line with sane defaults.
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

// progressOutput() displays a useful progress bar in the terminal while the
// frames are being processed. We want to add color but then we'd be using a
// third party dependency.... yikes .. just ... yikes. I don't know what to do
// dear God help me. HELP MEEEE I DONT DESERVE THIS! I'M A GOOD PERSON NOT THE
// SERVANT OF THE FUCKING DEVIL (joke)
func progressOutput(i int, totalPngFrames time.Duration) {
	// Clear the terminal (this is what makes it look animated).
	clearTerm()

	// This is a little heading to let us know everything is gonna be okay,
	// I'm here now.
	fmt.Printf("\nCreating %d frames from %d days of market data:\n\n",
		int(totalPngFrames), int(days))

	// This voodoo determines the progress in terms of percent. It looks
	// ridiculous to me, but I wrote it and usually never write incorrect
	// code but this just doesn't look right. I must've been high on LSD.
	percentDone := int(math.Floor(((1.00/(float64(totalPngFrames)/
		(float64(i)+0.01)))*100.00)*100) / 100)

	// No idea why this is needed. Will remove at some point for
	// insemination and find out what its deal is. We don't allow extra
	// lines of code that do nothing in this code base, or this code base
	// would be nothing.
	if percentDone >= 100 {
		percentDone = 100
	}

	// loading will be a progress bar, displayed to the user as terminal
	// output. blanks is used for formatting, providing padding for the
	// progress bar so the text doesn't shift. People notice bad
	// formatting! People remember and judge you forever on your
	// formatting, and formatting can effect the outcome of your entire
	// life. My formatting is better than what most US courts require, and
	// achieving a level of formatting this prestigious took me almost no
	// effort. Maybe he's born with it, maybe its maybleen.
	loading, blanks := "", ""

	// Make the loading bar using percentDone to determine the iteration
	// count, and thus, the number of characters that will be appended to
	// loading to visually represent what many know as a "progress bar".
	for q := 0; q <= int(percentDone)/6; q++ {
		loading = loading + "="
	}

	// Make the padding by adding spaces to the blanks var. This keeps the
	// text from sliding around:
	//
	//   [=========            ]    46%
	//        ^          ^           ^
	//     loading     blanks       text
	//
	for y := 0; y <= (100/6)-len(loading); y++ {
		if len(loading) <= 100/6 {
			blanks = blanks + " "
		} else {
			// May or may not be necessary, we really don't know.
			blanks = ""
		}
	}

	// Finally, we print a neatly formatted string, indicating the progress
	// with the nifty progress bar elements conjured above, and all it took
	// was the belief that it was possible and the will to make it happen.
	// YOU ARE THE MAGIC! GG MOTHER FUCKER!
	fmt.Printf(" [%s %s %d%%]  --  frame %d of %d\n\n",
		loading, blanks, percentDone, i, int(totalPngFrames))
}

// clearTerm() is used to clear the terminal when updating the progress bar
// displayed by progressBarOutput(). To do this it runs fmt.Println() 200
// times, but this number is excessive, and could be reduced. Also the function
// could take an arg to determine this number, and it probably should. Not
// important enough to fix though, so it remains suspended in a state of limbo.
func clearTerm() {
	for i := 0; i <= 200; i++ {
		fmt.Println()
	}
}

// parseQuotes() takes the file containing our data set, which, at the time of
// this writing, is simply a single .csv file containing historical, day-scale
// time-series data on bitcoin, splitting the string representation of this
// file by each new line, and returning them as a []string slice, each line
// containing un-refined quote data.
func parseQuotes(quoteFile string) (s [][]string) {
	b, err := os.ReadFile(quoteFile)
	if err != nil {
		log.Println(err)
	}
	lines := strings.Split(string(b), "\n")[1:]
	for _, l := range lines {
		s = append(s, strings.Split(l, ","))
	}
	return
}

// mkQuotes() takes the slice of unprocessed quotes returned by parseQuotes()
// and processes it, transforming it into our *quote type data structure, and
// creating structured data we can then use for deep analysis, allowing us to
// acquire hidden insights and valuable wisdoms that many spend their whole
// lives seeking, but few will ever know. We also use a map to map a time.Time
// to each *quote, creating a handy dandy, useful and convenient, nifty little
// pocket-size go-anywhere do anything time-series in a [next word here]. We
// cycle through the lines, parsing the quote data accordingly, and finally,
// adding it to the map:
func mkQuotes(quotes_ [][]string) {
	for _, q := range quotes_ {
		if d, err := time.Parse(time.DateOnly, q[0]); err == nil {
			qm[d] = map[string]float64{
				"End":       getF64(q[2]),
				"Volume":    getF64(q[6]),
				"MarketCap": getF64(q[7]),
				"Date":      float64(d.UnixMicro()),
			}
		}
	}
}

// getF64() parses a float64 type from a string and returns it.
func getF64(q string) float64 {
	price, err := strconv.ParseFloat(q, 64)
	if err != nil {
		log.Println(err)
	}
	return price
}

// newPlot() is a helper function that reduces the code base/typing a little.
// There may be other default options one could add here in future versions of
// bplot.
func newPlot() *plot.Plot {
	p := plot.New()
	p.Add(plotter.NewGrid())
	p.HideAxes()

	// Here we format the float to 2 decimal places so that its not
	// atrociously long.
	s := strconv.FormatFloat(qm[et]["End"], 'f', 2, 64)

	// Style the graph
	p.Title.Text = et.Format(title_txt) + s + " USD"
	p.Title.Padding = 50
	p.Title.TextStyle.Font.Variant = "Mono"
	p.Title.TextStyle.Font.Size = 25
	p.Title.TextStyle.YAlign = -1.5
	// p.Title.TextStyle.XAlign = 0

	return p
}
