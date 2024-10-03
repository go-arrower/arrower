package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"

	"github.com/go-arrower/arrower"
	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/app"
	admin_init "github.com/go-arrower/arrower/contexts/admin/init"
	"github.com/go-arrower/arrower/contexts/auth"
	auth_init "github.com/go-arrower/arrower/contexts/auth/init"
)

func main() {
	ctx, _ := context.WithCancel(context.Background())

	arrower, shutdown, err := arrower.New()
	if err != nil {
		panic(err)
	}

	//err = arrower.Settings.Save(ctx, alog.SettingLogLevel, setting.NewValue(int(slog.LevelDebug)))
	//alog.Unwrap(arrower.Logger).SetLevel(slog.LevelDebug)
	alog.Unwrap(arrower.Logger).SetLevel(alog.LevelDebug)

	//
	// load and initialise optional contexts provided by arrower
	adminContext, err := admin_init.NewAdminContext(ctx, arrower)
	if err != nil {
		panic(err)
	}

	authContext, err := auth_init.NewAuthContext(arrower)
	if err != nil {
		panic(err)
	}

	//
	// example route for a simple one-file setup
	arrower.WebRouter.GET("/", func(c echo.Context) error {
		sess, err := session.Get(auth.SessionName, c)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		userID := "World"
		if id, ok := sess.Values[auth.SessKeyUserID].(string); ok {
			userID = id
		}

		flashes := sess.Flashes()

		err = sess.Save(c.Request(), c.Response())
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		p := map[string]interface{}{
			"Title":   "Welcome to Arrower!",
			"userID":  userID,
			"Flashes": flashes,
			"UserID":  userID,
		}

		return c.Render(http.StatusOK, "=>home", p)
	})

	arrower.ArrowerQueue.RegisterJobFunc(func(ctx context.Context, j ExampleCron) error {
		arrower.Logger.InfoContext(ctx, "")
		arrower.Logger.InfoContext(ctx, "", slog.Any("name", j.Name))
		arrower.Logger.InfoContext(ctx, "")
		return nil
	})

	arrower.ArrowerQueue.Schedule("@every 1m", ExampleCron{"EX 000"})
	arrower.ArrowerQueue.Schedule("@every 1m", ExampleCron{"EX 111"})

	//
	// start app
	// initRegularExampleQueueLoad(ctx, arrower)
	arrower.WebRouter.Logger.Fatal(arrower.WebRouter.Start(fmt.Sprintf(":%d", arrower.Config.HTTP.Port)))

	//
	// shutdown app
	// todo implement graceful shutdown ect
	_ = shutdown(ctx)
	_ = authContext.Shutdown(ctx)
	_ = adminContext.Shutdown(ctx)
}

func initRegularExampleQueueLoad(ctx context.Context, di *arrower.Container) {
	type (
		SomeJob        struct{}
		LongRunningJob struct{}
	)

	_ = di.DefaultQueue.RegisterJobFunc(
		func(ctx context.Context, job SomeJob) error {
			di.Logger.InfoContext(ctx, "LOG ASYNC SIMPLE JOB")
			//panic("SOME JOB PANICS")

			time.Sleep(time.Duration(rand.Intn(10)) * time.Second) //nolint:gosec,mnd // weak numbers are ok, it is wait time

			if rand.Intn(100) > 30 { //nolint:gosec,mnd
				return errors.New("some error") //nolint:goerr113
			}

			return nil
		},
	)

	_ = di.DefaultQueue.RegisterJobFunc(
		app.NewInstrumentedJob[NamedJob](di.TraceProvider, di.MeterProvider, di.Logger, &namedJobHandler{Logger: di.Logger}).H,
	)

	_ = di.DefaultQueue.RegisterJobFunc(
		func(ctx context.Context, job LongRunningJob) error {
			time.Sleep(time.Duration(rand.Intn(5)) * time.Minute) //nolint:gosec,mnd // weak numbers are ok, it is wait time

			if rand.Intn(100) > 95 { //nolint:gosec,mnd
				return errors.New("some error") //nolint:goerr113
			}

			return nil
		},
	)

	go func() {
		ticker := time.NewTicker(10 * time.Second)

		for {
			select {
			case <-ticker.C:
				r := rand.Intn(100)

				if r%5 == 0 {
					_ = di.DefaultQueue.Enqueue(ctx, SomeJob{})
				}

				if r%12 == 0 {
					for i := 0; i/2 < r; i++ {
						// for i := range r { // fixme use new go1.22 style
						_ = di.DefaultQueue.Enqueue(ctx, NamedJob{Name: gofakeit.Name()})
					}
				}

				if r == 0 {
					_ = di.DefaultQueue.Enqueue(ctx, LongRunningJob{})
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

type ExampleCron struct {
	Name string
}

type NamedJob struct{ Name string }
type namedJobHandler struct {
	Logger alog.Logger
}

func (h *namedJobHandler) H(ctx context.Context, job NamedJob) error {
	h.Logger.InfoContext(ctx, "named job", slog.String("name", job.Name))

	time.Sleep(time.Duration(rand.Intn(4)) * time.Second)

	return nil
}
