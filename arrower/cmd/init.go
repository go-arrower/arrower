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
	"github.com/go-arrower/arrower/arrower/internal/initialise"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "EXPERIMENTAL! Initialises a blank arrower project",
		Long:  ``,
		Args: func(cmd *cobra.Command, args []string) error {
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
			yellow := color.New(color.FgYellow, color.Bold).FprintlnFunc()

			yellow(cmd.OutOrStdout(), "Existing folder content is ignored")

			projectName := strings.TrimSpace(args[0])
			data := projectData{
				Name:        projectName,
				NameSmall:   strings.ToLower(projectName),
				ProjectPath: strings.TrimSpace(args[1]),
				NameCaps:    strings.ToUpper(projectName),
			}

			addFile("README.md.templ", data)
			addFile("Makefile.templ", data)
			addFile("package.json.templ", data)
			addFile(".gitignore", data)

			os.MkdirAll(".config/githooks/", 0o755)
			addFile(".config/tailwind.config.js", data)
			addFile(".config/golangci.yaml.templ", data)
			addFile(".config/project.config.yaml.templ", data)
			addFile(".config/project_test.config.yaml.templ", data)
			addFile(".config/eslint.config.js", data)
			addFile(".config/001_start-containers.hook.go", data)
			addFile(".config/002_rebuild-tailwind-css.hook.go", data)
			addFile(".config/.prettierrc", data)
			addFile(".config/.prettierignore", data)
			addFile(".config/githooks/commit-msg", data)
			addFile(".config/githooks/pre-commit", data)

			if strings.Contains(data.ProjectPath, "github.com") {
				os.MkdirAll(".github/workflows", 0o755)
				addFile(".github/dependabot.yaml", data)
				addFile(".github/workflows/continuous-integration.yaml", data)
				addFile(".github/workflows/weekly-update.yaml", data)
			}

			os.MkdirAll("cmd", 0o755)
			addFile("cmd/root.go.templ", data)
			addFile("cmd/project.cmd.go.templ", data)
			addFile("cmd/project.cmd_test.go.templ", data)
			addFile("main.go.templ", data)
			addFile("go.mod.templ", data)

			os.MkdirAll("shared/infrastructure/config", 0o755)
			addFile("shared/infrastructure/config/config.go.templ", data)

			os.MkdirAll("shared/init", 0o755)
			addFile("shared/init/init.go.templ", data)
			addFile("shared/init/routes.go.templ", data)

			os.MkdirAll("shared/infrastructure/postgres/migrations", 0o755)
			copy("shared/infrastructure/postgres/migrations.go")
			copy("shared/infrastructure/postgres/migrations/000001_create_schema_arrower.down.sql")
			copy("shared/infrastructure/postgres/migrations/000001_create_schema_arrower.up.sql")
			copy("shared/infrastructure/postgres/migrations/000002_create_updated_at_trigger.down.sql")
			copy("shared/infrastructure/postgres/migrations/000002_create_updated_at_trigger.up.sql")
			copy("shared/infrastructure/postgres/migrations/000003_create_gue_jobs_table.down.sql")
			copy("shared/infrastructure/postgres/migrations/000003_create_gue_jobs_table.up.sql")
			copy("shared/infrastructure/postgres/migrations/000004_create_log.down.sql")
			copy("shared/infrastructure/postgres/migrations/000004_create_log.up.sql")
			copy("shared/infrastructure/postgres/migrations/000005_create_setting.down.sql")
			copy("shared/infrastructure/postgres/migrations/000005_create_setting.up.sql")

			os.MkdirAll("shared/domain", 0o755)
			os.MkdirAll("shared/application", 0o755)
			os.MkdirAll("shared/interfaces/repository", 0o755)

			os.MkdirAll("shared/interfaces/web", 0o755)
			addFile("shared/interfaces/web/project.controller.go.templ", data)

			os.MkdirAll("public/js/modules", 0o755)
			os.MkdirAll("public/css", 0o755)
			os.MkdirAll("public/icons", 0o755)
			copy("public/icons/96x96.png")
			copy("public/icons/android-chrome-192x192.png")
			copy("public/icons/android-chrome-512x512.png")
			copy("public/icons/apple-touch-icon.png")
			copy("public/icons/favicon.ico")

			addFile("public/manifest.json.templ", data)
			copy("public/assets.go")

			os.MkdirAll("shared/views/pages", 0o755)
			addFile("shared/views/pages/home.html.templ", data)
			addFile("shared/views/views.go.templ", data)
			addFile("shared/views/input.css", data)
			addFile("shared/views/default.base.html.templ", data)

			os.MkdirAll("devops/grafana", 0o755)
			addFile("devops/grafana/datasource.yaml.templ", data)

			os.MkdirAll("devops/pgadmin", 0o755)
			addFile("devops/pgadmin/servers.json.templ", data)
			addFile("devops/pgadmin/pgpass.templ", data)

			os.MkdirAll("devops/prometheus", 0o755)
			copy("devops/prometheus/prometheus.yaml")

			os.MkdirAll("devops/tempo", 0o755)
			copy("devops/tempo/tempo.yaml")

			addFile("devops/docker-compose.yaml.templ", data)

			os.MkdirAll("tests/e2e/cypress/e2e", 0o755)
			os.MkdirAll("tests/e2e/cypress/fixtures", 0o755)
			os.MkdirAll("tests/e2e/cypress/screenshots", 0o755)
			os.MkdirAll("tests/e2e/cypress/support", 0o755)
			os.MkdirAll("tests/e2e/cypress/videos", 0o755)
			copy("tests/e2e/cypress.config.js")
			copy("tests/e2e/cypress/e2e/status.cy.js")
			copy("tests/e2e/cypress/support/commands.js")
			copy("tests/e2e/cypress/support/e2e.js")

			yellow(cmd.OutOrStdout(), "Project initialised. Download dependencies")
			acmd := exec.Command("go", "mod", "tidy")
			acmd.Stdout = cmd.OutOrStdout()
			acmd.Stderr = cmd.OutOrStderr()
			acmd.Run()
			acmd = exec.Command("go", "mod", "download")
			acmd.Stdout = cmd.OutOrStdout()
			acmd.Stderr = cmd.OutOrStderr()
			acmd.Run()
			acmd = exec.Command("npm", "install", "--package-lock-only")
			acmd.Stdout = cmd.OutOrStdout()
			acmd.Stderr = cmd.OutOrStderr()
			acmd.Run()
			acmd = exec.Command("make", "dev-tools")
			acmd.Stdout = cmd.OutOrStdout()
			acmd.Stderr = cmd.OutOrStderr()
			acmd.Run()
			acmd = exec.Command("make", "generate")
			acmd.Stdout = cmd.OutOrStdout()
			acmd.Stderr = cmd.OutOrStderr()
			acmd.Run()

			blue(cmd.OutOrStdout(), "Initialisation complete")
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
		return fmt.Errorf("could not execute template: %v", err)
	}

	targetFileName := strings.TrimSuffix(templName, ".templ")
	targetFileName = strings.ReplaceAll(targetFileName, "project", data.NameSmall)

	if err := os.WriteFile(path.Join(".", targetFileName), buf.Bytes(), os.ModePerm); err != nil {
		return fmt.Errorf("could not write file: %w", err)
	}

	return nil
}

func copy(src string) error {
	source, err := initialise.TemplatesFS.ReadFile("templates/" + src)
	if err != nil {
		return err
	}

	return os.WriteFile(path.Join(".", src), source, 0o644)
}
