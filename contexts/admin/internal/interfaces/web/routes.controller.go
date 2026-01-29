package web

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
)

func NewRoutesController(echo *echo.Echo) *RoutesController {
	return &RoutesController{echo: echo}
}

type RoutesController struct {
	echo *echo.Echo
}

func (ctrl *RoutesController) Index() func(e echo.Context) error {
	type route struct {
		*echo.Route
		ColourCount int
		Colour      string
		HasParams   bool
	}

	return func(c echo.Context) error {
		echoRoutes := ctrl.echo.Routes()
		routes := make([]*route, len(echoRoutes))

		// sort routes by path and then by method
		sort.Slice(echoRoutes, func(i, j int) bool {
			if echoRoutes[i].Path < echoRoutes[j].Path {
				return true
			}

			if echoRoutes[i].Path == echoRoutes[j].Path {
				return echoRoutes[i].Method < echoRoutes[j].Method
			}

			return false
		})

		var (
			lastPathPrefix string
			group          int
		)

		// assign a colour group to all routs sharing the same initial path prefix
		for i, r := range echoRoutes {
			pathPrefix := strings.Split(echoRoutes[i].Path, "/")[1]
			if lastPathPrefix != pathPrefix {
				lastPathPrefix = pathPrefix
				group++
			}

			routes[i] = &route{r, group, "", strings.Contains(r.Path, ":")}
		}

		// colours := generateColorsVibrant(group + 1)
		colours := generateColorsMuted(group + 1)

		for i := range echoRoutes {
			routes[i].Colour = colours[routes[i].ColourCount]
		}

		return c.Render(http.StatusOK, "admin.routes", echo.Map{
			"Flashes": nil,
			"Routes":  routes,
		})
	}
}

func hslToHex(h, s, l float64) string {
	h = h / 360.0

	var r, g, b float64

	if s == 0 {
		r, g, b = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q

		r = hueToRGB(p, q, h+1.0/3.0)
		g = hueToRGB(p, q, h)
		b = hueToRGB(p, q, h-1.0/3.0)
	}

	return rgbToHex(
		uint8(math.Round(r*255)),
		uint8(math.Round(g*255)),
		uint8(math.Round(b*255)),
	)
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

func rgbToHex(r, g, b uint8) string {
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func generateColorsMuted(n int) []string {
	colors := make([]string, n)

	baseHues := []float64{
		210, // Blue
		0,   // Red
		150, // Green
		280, // Purple
		30,  // Orange
		180, // Cyan
		330, // Rose
		45,  // Brown
	}

	for i := 0; i < n; i++ {
		hue := baseHues[i%len(baseHues)]
		saturation := 0.5 + float64(i%3)*0.05
		lightness := 0.32 + float64(i%2)*0.03

		colors[i] = hslToHex(hue, saturation, lightness)
	}

	return colors
}

func generateColorsVibrant(n int) []string {
	colors := make([]string, n)

	baseHues := []float64{
		0,   // Red
		210, // Blue
		120, // Green
		280, // Purple
		30,  // Orange
		180, // Cyan
		300, // Magenta
		60,  // Yellow
		330, // Pink
		150, // Teal
	}

	for i := 0; i < n; i++ {
		hue := baseHues[i%len(baseHues)]
		saturation := 0.8 + float64(i%2)*0.05
		lightness := 0.35 + float64(i%3)*0.03

		colors[i] = hslToHex(hue, saturation, lightness)
	}

	return colors
}
