// misspell: external library uses "color" (American spelling), not "colour"
// err113: dynamic over sentinel errors is alright for CLI package (no package API)
//
//nolint:misspell,err113,mnd
package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/go-arrower/arrower/arrower/internal/initialise"
)

const (
	dirPerm  = 0o755
	filePerm = 0o644
)

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "EXPERIMENTAL! Initialises a blank arrower project",
		Long:  ``,
		//nolint:mnd
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("requires a project name")
			}

			if len(args) < 2 {
				return errors.New("requires a import path")
			}

			return nil
		},
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			blue := color.New(color.FgBlue, color.Bold).FprintlnFunc()
			green := color.New(color.FgGreen, color.Bold).FprintlnFunc()
			yellow := color.New(color.FgYellow, color.Bold).FprintlnFunc()
			red := color.New(color.FgRed, color.Bold).FprintlnFunc()

			yellow(cmd.OutOrStdout(), "Existing folder content is ignored")

			projectName := strings.Title(strings.TrimSpace(args[0]))
			data := projectData{
				Name:        projectName,
				NameSmall:   strings.ToLower(projectName),
				ProjectPath: strings.TrimSpace(args[1]),
				NameCaps:    strings.ToUpper(projectName),
			}

			var err error

			err = errors.Join(err, addFile("README.md.templ", data))
			err = errors.Join(err, addFile("Makefile.templ", data))
			err = errors.Join(err, addFile("package.json.templ", data))
			err = errors.Join(err, addFile(".gitignore", data))

			err = errors.Join(err, os.MkdirAll(".config/githooks/", dirPerm))
			err = errors.Join(err, addFile(".config/tailwind.config.js", data))
			err = errors.Join(err, addFile(".config/golangci.yaml.templ", data))
			err = errors.Join(err, addFile(".config/project.config.yaml.templ", data))
			err = errors.Join(err, addFile(".config/project_test.config.yaml.templ", data))
			err = errors.Join(err, addFile(".config/eslint.config.js", data))
			err = errors.Join(err, addFile(".config/001_start-containers.hook.go", data))
			err = errors.Join(err, addFile(".config/002_rebuild-tailwind-css.hook.go", data))
			err = errors.Join(err, addFile(".config/.prettierrc", data))
			err = errors.Join(err, addFile(".config/.prettierignore", data))
			err = errors.Join(err, addFile(".config/githooks/commit-msg", data))
			err = errors.Join(err, addFile(".config/githooks/pre-commit", data))

			if strings.Contains(data.ProjectPath, "github.com") {
				err = errors.Join(err, os.MkdirAll(".github/workflows", dirPerm))
				err = errors.Join(err, addFile(".github/dependabot.yaml", data))
				err = errors.Join(err, addFile(".github/workflows/continuous-integration.yaml", data))
				err = errors.Join(err, addFile(".github/workflows/weekly-update.yaml", data))
			}

			err = errors.Join(err, os.MkdirAll("cmd", dirPerm))
			err = errors.Join(err, addFile("cmd/root.go.templ", data))
			err = errors.Join(err, addFile("cmd/project.cmd.go.templ", data))
			// err = errors.Join(err, addFile("cmd/project.cmd_test.go.templ", data)) // TODO create test file
			err = errors.Join(err, addFile("main.go.templ", data))
			err = errors.Join(err, addFile("go.mod.templ", data))

			err = errors.Join(err, os.MkdirAll("shared/infrastructure/config", dirPerm))
			err = errors.Join(err, addFile("shared/infrastructure/config/config.go.templ", data))

			err = errors.Join(err, os.MkdirAll("shared/init", dirPerm))
			err = errors.Join(err, addFile("shared/init/init.go.templ", data))
			err = errors.Join(err, addFile("shared/init/routes.go.templ", data))

			err = errors.Join(err, os.MkdirAll("shared/infrastructure/postgres/migrations", dirPerm))
			err = errors.Join(err, cp("shared/infrastructure/postgres/migrations.go"))
			err = errors.Join(err, cp("shared/infrastructure/postgres/migrations/000001_create_schema_arrower.down.sql"))
			err = errors.Join(err, cp("shared/infrastructure/postgres/migrations/000001_create_schema_arrower.up.sql"))
			err = errors.Join(err, cp("shared/infrastructure/postgres/migrations/000002_create_updated_at_trigger.down.sql"))
			err = errors.Join(err, cp("shared/infrastructure/postgres/migrations/000002_create_updated_at_trigger.up.sql"))
			err = errors.Join(err, cp("shared/infrastructure/postgres/migrations/000003_create_gue_jobs_table.down.sql"))
			err = errors.Join(err, cp("shared/infrastructure/postgres/migrations/000003_create_gue_jobs_table.up.sql"))
			err = errors.Join(err, cp("shared/infrastructure/postgres/migrations/000004_create_log.down.sql"))
			err = errors.Join(err, cp("shared/infrastructure/postgres/migrations/000004_create_log.up.sql"))
			err = errors.Join(err, cp("shared/infrastructure/postgres/migrations/000005_create_setting.down.sql"))
			err = errors.Join(err, cp("shared/infrastructure/postgres/migrations/000005_create_setting.up.sql"))

			err = errors.Join(err, os.MkdirAll("shared/domain", dirPerm))
			err = errors.Join(err, os.MkdirAll("shared/application", dirPerm))
			err = errors.Join(err, os.MkdirAll("shared/interfaces/repository", dirPerm))

			err = errors.Join(err, os.MkdirAll("shared/interfaces/web", dirPerm))
			err = errors.Join(err, addFile("shared/interfaces/web/project.controller.go.templ", data))

			err = errors.Join(err, os.MkdirAll("public/js/modules", dirPerm))
			err = errors.Join(err, os.MkdirAll("public/css", dirPerm))
			err = errors.Join(err, os.MkdirAll("public/icons", dirPerm))
			err = errors.Join(err, cp("public/icons/96x96.png"))
			err = errors.Join(err, cp("public/icons/android-chrome-192x192.png"))
			err = errors.Join(err, cp("public/icons/android-chrome-512x512.png"))
			err = errors.Join(err, cp("public/icons/apple-touch-icon.png"))
			err = errors.Join(err, cp("public/icons/favicon.ico"))

			err = errors.Join(err, addFile("public/manifest.json.templ", data))
			err = errors.Join(err, cp("public/assets.go"))

			err = errors.Join(err, os.MkdirAll("shared/views/pages", dirPerm))
			err = errors.Join(err, addFile("shared/views/pages/home.html.templ", data))
			err = errors.Join(err, addFile("shared/views/views.go.templ", data))
			err = errors.Join(err, addFile("shared/views/input.css", data))
			err = errors.Join(err, addFile("shared/views/default.base.html.templ", data))

			err = errors.Join(err, os.MkdirAll("devops/grafana", dirPerm))
			err = errors.Join(err, addFile("devops/grafana/datasource.yaml.templ", data))

			err = errors.Join(err, os.MkdirAll("devops/pgadmin", dirPerm))
			err = errors.Join(err, addFile("devops/pgadmin/servers.json.templ", data))
			err = errors.Join(err, addFile("devops/pgadmin/pgpass.templ", data))

			err = errors.Join(err, os.MkdirAll("devops/prometheus", dirPerm))
			err = errors.Join(err, cp("devops/prometheus/prometheus.yaml"))

			err = errors.Join(err, os.MkdirAll("devops/tempo", dirPerm))
			err = errors.Join(err, cp("devops/tempo/tempo.yaml"))

			err = errors.Join(err, addFile("devops/docker-compose.yaml.templ", data))

			err = errors.Join(err, os.MkdirAll("tests/e2e/cypress/e2e", dirPerm))
			err = errors.Join(err, os.MkdirAll("tests/e2e/cypress/fixtures", dirPerm))
			err = errors.Join(err, os.MkdirAll("tests/e2e/cypress/screenshots", dirPerm))
			err = errors.Join(err, os.MkdirAll("tests/e2e/cypress/support", dirPerm))
			err = errors.Join(err, os.MkdirAll("tests/e2e/cypress/videos", dirPerm))
			err = errors.Join(err, cp("tests/e2e/cypress.config.js"))
			err = errors.Join(err, cp("tests/e2e/cypress/e2e/status.cy.js"))
			err = errors.Join(err, cp("tests/e2e/cypress/support/commands.js"))
			err = errors.Join(err, cp("tests/e2e/cypress/support/e2e.js"))
			if err != nil {
				red(cmd.OutOrStdout(), "Failed to initialise project! Could not copy Arrower")

				return fmt.Errorf("%w", err)
			}

			yellow(cmd.OutOrStdout(), "Project initialised. Download dependencies")

			acmd := exec.CommandContext(cmd.Context(), "go", "mod", "tidy")
			acmd.Stdout = cmd.OutOrStdout()
			acmd.Stderr = cmd.OutOrStderr()
			err = errors.Join(nil, acmd.Run())
			acmd = exec.CommandContext(cmd.Context(), "go", "mod", "download")
			acmd.Stdout = cmd.OutOrStdout()
			acmd.Stderr = cmd.OutOrStderr()
			err = errors.Join(err, acmd.Run())
			acmd = exec.CommandContext(cmd.Context(), "npm", "install", "--package-lock-only")
			acmd.Stdout = cmd.OutOrStdout()
			acmd.Stderr = cmd.OutOrStderr()
			err = errors.Join(err, acmd.Run())
			acmd = exec.CommandContext(cmd.Context(), "make", "dev-tools")
			acmd.Stdout = cmd.OutOrStdout()
			acmd.Stderr = cmd.OutOrStderr()
			err = errors.Join(err, acmd.Run())
			acmd = exec.CommandContext(cmd.Context(), "make", "generate")
			acmd.Stdout = cmd.OutOrStdout()
			acmd.Stderr = cmd.OutOrStderr()
			err = errors.Join(err, acmd.Run())

			acmd = exec.CommandContext(cmd.Context(), "git", "init", ".", "-b", "master")
			acmd.Stdout = cmd.OutOrStdout()
			acmd.Stderr = cmd.OutOrStderr()
			err = errors.Join(err, acmd.Run())
			acmd = exec.CommandContext(cmd.Context(), "git", "add", ".")
			acmd.Stdout = cmd.OutOrStdout()
			acmd.Stderr = cmd.OutOrStderr()
			err = errors.Join(err, acmd.Run())
			if err != nil {
				red(cmd.OutOrStdout(), "Failed to initialise project! Could not finish initialisation")

				return fmt.Errorf("%w", err)
			}

			green(cmd.OutOrStdout(), "Initialisation complete")
			blue(cmd.OutOrStdout(), "Run `git commit -m \"chore: initial Arrower application\"`")
			blue(cmd.OutOrStdout(), "Run `make run`")

			return nil
		},
	}

	return cmd
}

type projectData struct {
	Name        string
	NameSmall   string
	ProjectPath string
	NameCaps    string
}

func addFile(templName string, data projectData) error {
	tmpl, err := template.ParseFS(initialise.TemplatesFS, "templates/"+templName)
	if err != nil {
		return fmt.Errorf("could not parse template FS: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}

	targetFileName := strings.TrimSuffix(templName, ".templ")
	targetFileName = strings.ReplaceAll(targetFileName, "project", data.NameSmall)

	if err := os.WriteFile(path.Join(".", targetFileName), buf.Bytes(), filePerm); err != nil {
		return fmt.Errorf("could not write file: %w", err)
	}

	return nil
}

func cp(src string) error {
	source, err := initialise.TemplatesFS.ReadFile("templates/" + src)
	if err != nil {
		return fmt.Errorf("failed to read template file %q: %w", src, err)
	}

	return os.WriteFile(path.Join(".", src), source, filePerm)
}
