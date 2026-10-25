package mail

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/flosch/pongo2/v7"
	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/db"
)

//go:embed templates/*
var mailFS embed.FS

type MailViews struct {
	Set *pongo2.TemplateSet
}

func NewMailViews(debug bool) (*MailViews, error) {
	set, err := loadSet("templates", mailFS, debug)
	if err != nil {
		return nil, err
	}

	return &MailViews{Set: set}, nil
}

func (v *MailViews) RenderEmailFirstPhaseView(runtime configuration.RuntimeEnv, onboardTeamEmail []string, reg *db.RegistrationRecord) (string, error) {

	ctx := pongo2.Context{
		"Runtime":          runtime,
		"OnboardTeamEmail": getOnboardTeamEmail(onboardTeamEmail),
		"Reg":              reg,
	}

	return executeTemplate(v.Set, "email_first_phase.html.j2", ctx)
}

func (v *MailViews) RenderEmailSecondPhaseView(runtime configuration.RuntimeEnv, onboardTeamEmail []string, reg *db.RegistrationRecord) (string, error) {

	ctx := pongo2.Context{
		"Runtime":          runtime,
		"OnboardTeamEmail": getOnboardTeamEmail(onboardTeamEmail),
		"Reg":              reg,
	}

	return executeTemplate(v.Set, "email_second_phase.html.j2", ctx)
}

func (v *MailViews) RenderReminderEmailView(runtime configuration.RuntimeEnv, onboardTeamEmail []string, reg *db.RegistrationRecord) (string, error) {

	ctx := pongo2.Context{
		"Runtime":          runtime,
		"OnboardTeamEmail": getOnboardTeamEmail(onboardTeamEmail),
		"Reg":              reg,
	}

	return executeTemplate(v.Set, "email_reminder.html.j2", ctx)
}

func (v *MailViews) RenderVerificationCodeEmailView(runtime configuration.RuntimeEnv, onboardTeamEmail []string, code string) (string, error) {

	ctx := pongo2.Context{
		"Runtime":          runtime,
		"OnboardTeamEmail": getOnboardTeamEmail(onboardTeamEmail),
		"Code":             code,
	}

	return executeTemplate(v.Set, "email_verification.html.j2", ctx)
}

func (v *MailViews) RenderTestEmailView(runtime configuration.RuntimeEnv, onboardTeamEmail []string, code string) (string, error) {

	ctx := pongo2.Context{
		"Runtime":          runtime,
		"OnboardTeamEmail": getOnboardTeamEmail(onboardTeamEmail),
		"Code":             code,
	}

	return executeTemplate(v.Set, "email_test.html.j2", ctx)
}
func (v *MailViews) RenderCheckIssuerErrorView(runtime configuration.RuntimeEnv, errorMsg string, payload string) (string, error) {

	fmt.Printf("Enviroment: %v\n", runtime)
	ctx := pongo2.Context{
		"Runtime":  runtime,
		"ErrorMsg": errorMsg,
		"Payload":  payload,
	}

	return executeTemplate(v.Set, "check_issuer_error.html.j2", ctx)
}

func (v *MailViews) RenderIssuerErrorView(runtime configuration.RuntimeEnv, errorMsg string, payload string, reg *db.RegistrationRecord) (string, error) {

	fmt.Printf("Enviroment: %v\n", runtime)
	ctx := pongo2.Context{
		"Runtime":  runtime,
		"Reg":      reg,
		"ErrorMsg": errorMsg,
		"Payload":  payload,
	}

	return executeTemplate(v.Set, "issuer_error.html.j2", ctx)
}

func getOnboardTeamEmail(onboardTeamEmail []string) string {
	for _, email := range onboardTeamEmail {
		trimmed := strings.TrimSpace(email)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func loadSet(subapp string, embeddedFS embed.FS, debug bool) (*pongo2.TemplateSet, error) {
	var templateFS fs.FS
	if _, err := os.Stat(subapp); err == nil {
		templateFS = os.DirFS(subapp)
	} else if _, err := os.Stat("internal/mail/" + subapp); err == nil {
		templateFS = os.DirFS("internal/mail/" + subapp)
	} else {
		sub, err := fs.Sub(embeddedFS, subapp)
		if err != nil {
			return nil, err
		}
		templateFS = sub
	}

	loader := pongo2.NewFSLoader(templateFS)
	set := pongo2.NewSet(subapp, loader)
	set.Debug = debug

	set.Globals = pongo2.Context{
		"AppName": "DOME Marketplace",
	}

	return set, nil
}

func executeTemplate(set *pongo2.TemplateSet, templateName string, context pongo2.Context) (string, error) {
	tpl, err := set.FromCache(templateName)
	if err != nil {
		return "", err
	}

	out, err := tpl.Execute(context)
	if err != nil {
		return "", err
	}

	return out, nil
}
