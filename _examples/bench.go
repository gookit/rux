package main

import (
	"flag"
	"fmt"
	"github.com/gookit/sux"
	"math/rand"
	"strings"
	"time"
)

var (
	methods = strings.Split("GET,POST,PUT,PATCH,DELETE,OPTIONS,HEAD", ",")
	times = *flag.Int("times", 1000, "the match times")
	number = *flag.Int("number", 1000, "the generate routes numbers")
	percent = flag.Int("percent", 5, "the generate dynamic routes percentage. allow 1 - 10")
)

var r = sux.New()
var alphaNum = []byte(`0123456789abcdefghijklmnopqrstuvwxyz`)
var routes []map[string]string
var emptyHandler = func(c *sux.Context) {}
var (
	firstRoute, randRoute, lastRoute map[string]string
)

// simple benchmark testing
// go run ./_examples/bench.go
func main() {
	flag.Parse()

	if times < 1 {
		times = 100
	}

	if number < 100 {
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
	fmt.Printf(" - collect routes elapsed time: %.3f ms\n\n", time.Now().Sub(st).Seconds()*1000)
	fmt.Print("Begin route match\n\n")
	
	fmt.Print("- First route match ...")
	st = time.Now()
	matchRoute(times, firstRoute)
	total := time.Now().Sub(st).Seconds()*1000
	avg := total*1000/float64(times)
	fmt.Printf(" OK \n  总计耗时 %.3f ms, 匹配次数: %d . 平均耗时: %.3f us\n\n", total, times, avg)

	fmt.Print("- Random route match ...")
	st = time.Now()
	matchRoute(times, randRoute)
	total = time.Now().Sub(st).Seconds()*1000
	avg = total*1000/float64(times)
	fmt.Printf(" OK \n  总计耗时 %.3f ms, 匹配次数: %d . 平均耗时: %.3f us\n\n", total, times, avg)

	fmt.Print("- Last route match ...")
	st = time.Now()
	matchRoute(times, lastRoute)
	total = time.Now().Sub(st).Seconds()*1000
	avg = total*1000/float64(times)
	fmt.Printf(" OK \n  总计耗时 %.3f ms, 匹配次数: %d . 平均耗时: %.3f us\n\n", total, times, avg)

	fmt.Print("- Unknown route match ...")
	st = time.Now()
	matchRoute(times, map[string]string{"m": "GET", "p": "/not-exist"})
	total = time.Now().Sub(st).Seconds()*1000
	avg = total*1000/float64(times)
	fmt.Printf(" OK \n  总计耗时 %.3f ms, 匹配次数: %d . 平均耗时: %.3f us\n", total, times, avg)
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
	lastRoute = routes[number-1]
	randIndex := rand.Intn(number-1)
	randRoute = routes[randIndex]
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

func matchRoute(times int, item map[string]string) {
	for i := 0; i < times; i++ {
		r.Match(item["m"], item["p"])
	}
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
