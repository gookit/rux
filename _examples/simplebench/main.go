package main

import (
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gookit/rux"
)

var (
	methods  = strings.Split("GET,POST,PUT,PATCH,DELETE,OPTIONS,HEAD", ",")
	inTimes  = flag.Int("t", 1000, "the match times")
	inNumber = flag.Int("n", 1000, "the generate routes numbers")
	percent  = flag.Int("percent", 5, "the generate dynamic routes percentage. allow 1 - 10")
)

var r = rux.New()
var alphaNum = []byte(`0123456789abcdefghijklmnopqrstuvwxyz`)
var routes []map[string]string
var emptyHandler = func(c *rux.Context) {}
var (
	times, number                    int
	firstRoute, randRoute, lastRoute map[string]string
)

// simple benchmark testing
// go run ./_examples/bench.go
func main() {
	flag.Parse()

	number = *inNumber
	times = *inTimes

	if times < 1 {
		times = 100
	}

	if number < 5 {
		number = 100
	}

	fmt.Printf("Route Benchmark Testing\n\n")

	fmt.Print("Begin generate routes\n\n")
	generateRoutes()
	fmt.Printf(` - total generate routes: %d
 - first: %v
 - last: %v 
 - random: %v 
`, number, firstRoute, randRoute, lastRoute)

	st := time.Now()
	collectRoutes()
	fmt.Printf(" - collect routes elapsed time: %.3f ms\n\n", time.Since(st).Seconds()*1000)
	fmt.Print("Begin route match\n\n")

	fmt.Print("- First route match ...")
	st = time.Now()
	ok, route := matchRoute(times, firstRoute)
	total := time.Now().Sub(st).Seconds() * 1000
	avg := total * 1000 / float64(times)
	fmt.Printf(
		" OK \n  Total time consuming %.3f ms, Number of matches: (%d/%d). Average time consuming: %.3f us\n  Match result: %+v\n\n",
		total, ok, times, avg, route.Info(),
	)

	fmt.Print("- Random route match ...")
	st = time.Now()
	ok, route = matchRoute(times, randRoute)
	total = time.Now().Sub(st).Seconds() * 1000
	avg = total * 1000 / float64(times)
	fmt.Printf(
		" OK \n  Total time consuming %.3f ms, Number of matches: (%d/%d). Average time consuming: %.3f us\n  Match result: %+v\n\n",
		total, ok, times, avg, route.Info(),
	)

	fmt.Print("- Last route match ...")
	st = time.Now()
	ok, route = matchRoute(times, lastRoute)
	total = time.Now().Sub(st).Seconds() * 1000
	avg = total * 1000 / float64(times)
	fmt.Printf(
		" OK \n  Total time consuming %.3f ms, Number of matches: (%d/%d). Average time consuming: %.3f us\n  Match result: %+v\n\n",
		total, ok, times, avg, route.Info(),
	)

	fmt.Print("- Unknown route match ...")
	st = time.Now()
	_, route = matchRoute(times, map[string]string{"m": "GET", "p": "/not-exist"})
	total = time.Now().Sub(st).Seconds() * 1000
	avg = total * 1000 / float64(times)
	fmt.Printf(
		" OK \n  Total time consuming %.3f ms, Number of matches: %d. Average time consuming: %.3f us\n  Match result: %+v\n",
		total, times, avg, route.Info(),
	)
}

// generate routes
func generateRoutes() {
	routes = make([]map[string]string, number)
	for i := 0; i < number; i++ {
		routes[i] = map[string]string{
			"m": randomMethod(),
			"p": randomUrlPath(*percent),
		}
	}

	firstRoute = routes[0]
	firstRoute["p1"] = strings.NewReplacer("{", "f", "}", "f").Replace(firstRoute["p"])

	lastRoute = routes[number-1]
	lastRoute["p1"] = strings.NewReplacer("{", "l", "}", "l").Replace(lastRoute["p"])

	randIndex := rand.Intn(number - 1)
	randRoute = routes[randIndex]
	randRoute["p1"] = strings.NewReplacer("{", "r", "}", "r").Replace(randRoute["p"])
}

// collect routes
func collectRoutes() {
	for _, item := range routes {
		switch item["m"] {
		case "GET":
			r.GET(item["p"], emptyHandler)
		case "HEAD":
			r.HEAD(item["p"], emptyHandler)
		case "POST":
			r.POST(item["p"], emptyHandler)
		case "PUT":
			r.PUT(item["p"], emptyHandler)
		case "PATCH":
			r.PATCH(item["p"], emptyHandler)
		case "DELETE":
			r.DELETE(item["p"], emptyHandler)
		case "OPTIONS":
			r.OPTIONS(item["p"], emptyHandler)
		}
	}
}

func matchRoute(times int, item map[string]string) (ok int, rt *rux.Route) {
	ok = 0
	path := item["p1"]

	for i := 0; i < times; i++ {
		route,_,_ := r.Match(item["m"], path)
		if route != nil {
			ok++
		}
	}

	return
}

func randomMethod() string {
	// 每次提供不同的种子 不然生成的随机数会是一样的
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(methods))
	return methods[index]
}

func randomUrlPath(percent int) string {
	// 每次提供不同的种子 不然生成的随机数会是一样的
	rand.Seed(time.Now().UnixNano())

	nLen := len(alphaNum)
	charLen := rand.Intn(15) + 5
	path := "/"

	for i := 0; i < charLen; i++ {
		seed := time.Now().UnixNano()
		rand.Seed(seed)
		charIdx := rand.Intn(nLen)
		path += string(alphaNum[charIdx])

		rand.Seed(time.Now().UnixNano())
		if rand.Intn(10) == 1 {
			path += "/"
		}

		// add path var
		if strings.IndexByte(path, '{') == -1 {
			rand.Seed(time.Now().UnixNano())
			if rand.Intn(10) <= percent {
				path = strings.TrimRight(path, "/") + "/{name}/"
			}
		}
	}

	return strings.TrimRight(path, "/")
}
