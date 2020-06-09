package cli

import (
	"context"
	"fmt"
	"github.com/google/go-github/v32/github"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/pfnet-research/alertmanager-to-github/pkg/notifier"
	"github.com/pfnet-research/alertmanager-to-github/pkg/server"
	"github.com/pfnet-research/alertmanager-to-github/pkg/template"
	"golang.org/x/oauth2"
	"io/ioutil"
	"os"
)

const flagListen = "listen"
const flagGitHubEnterpriseURL = "github-enterprise-url"
const flagRepoOwner = "repo-owner"
const flagRepo = "repo"
const flagLabels = "labels"
const flagBodyTemplateFile = "body-template-file"
const flagTitleTemplateFile = "title-template-file"
const flagGitHubToken = "github-token"
const flagAlertIDTemplate = "alert-id-template"

func App() *cli.App {
	return &cli.App{
		Name:  os.Args[0],
		Usage: "Webhook receiver Alertmanager to create GitHub issues",
		Action: func(c *cli.Context) error {
			if err := action(c); err != nil {
				return cli.Exit(fmt.Errorf("error: %w", err), 1)
			}
			return nil
		},
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			if err != nil {
				log.Err(err).Msg("error")
			}
			return err
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  flagListen,
				Value: ":8080",
				Usage: "HTTP listen on",
			},
			&cli.StringFlag{
				Name:  flagGitHubEnterpriseURL,
				Usage: "GitHub Enterprise URL (e.g. https://github.example.com)",
			},
			&cli.StringFlag{
				Name:     flagRepoOwner,
				Required: true,
				Usage:    "Repository owner",
			},
			&cli.StringFlag{
				Name:     flagRepo,
				Required: true,
				Usage:    "Repository",
			},
			&cli.StringSliceFlag{
				Name:  flagLabels,
				Usage: "Issue labels",
			},
			&cli.StringFlag{
				Name:     flagBodyTemplateFile,
				Required: true,
				Usage:    "Body template file",
			},
			&cli.StringFlag{
				Name:     flagTitleTemplateFile,
				Required: true,
				Usage:    "Title template file",
			},
			&cli.StringFlag{
				Name:  flagAlertIDTemplate,
				Value: "{{.Payload.GroupKey}}",
				Usage: "Title template file",
			},
			&cli.StringFlag{
				Name:     flagGitHubToken,
				Required: true,
				EnvVars:  []string{"GITHUB_TOKEN"},
				Usage:    "GitHub API token (command line argument is not recommended)",
			},
		},
	}
}

func buildGitHubClient(githubURL string, token string) (*github.Client, error) {
	var err error
	var client *github.Client

	ctx := context.TODO()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	if githubURL == "" {
		client = github.NewClient(tc)
	} else {
		client, err = github.NewEnterpriseClient(githubURL, githubURL, tc)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func templateFromFile(path string) (*template.Template, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return templateFromString(string(b))
}

func templateFromString(s string) (*template.Template, error) {
	t, err := template.Parse(s)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func action(c *cli.Context) error {
	githubClient, err := buildGitHubClient(c.String(flagGitHubEnterpriseURL), c.String(flagGitHubToken))
	if err != nil {
		return err
	}

	bodyTemplate, err := templateFromFile(c.String(flagBodyTemplateFile))
	if err != nil {
		return err
	}

	titleTemplate, err := templateFromFile(c.String(flagTitleTemplateFile))
	if err != nil {
		return err
	}

	alertIDTemplate, err := templateFromString(c.String(flagAlertIDTemplate))
	if err != nil {
		return err
	}

	nt, err := notifier.NewGitHub()
	if err != nil {
		return err
	}
	nt.GitHubClient = githubClient
	nt.Repo = c.String(flagRepo)
	nt.Owner = c.String(flagRepoOwner)
	nt.Labels = c.StringSlice(flagLabels)
	nt.BodyTemplate = bodyTemplate
	nt.TitleTemplate = titleTemplate
	nt.AlertIDTemplate = alertIDTemplate

	router := server.New(nt).Router()
	if err := router.Run(c.String(flagListen)); err != nil {
		return err
	}

	return nil
}