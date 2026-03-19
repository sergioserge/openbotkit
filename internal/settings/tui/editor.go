package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/73ai/openbotkit/settings"
)

func buildForm(f *settings.Field, current string) (*huh.Form, *string, *bool) {
	var strVal string
	var boolVal bool

	switch f.Type {
	case settings.TypeSelect:
		strVal = current
		var opts []huh.Option[string]
		for _, o := range f.Options {
			opts = append(opts, huh.NewOption(o.Label, o.Value))
		}
		return huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(f.Label).
					Description(f.Description).
					Options(opts...).
					Value(&strVal),
			),
		), &strVal, nil

	case settings.TypeBool:
		boolVal = current == "true"
		return huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(f.Label).
					Description(f.Description).
					Value(&boolVal),
			),
		), nil, &boolVal

	case settings.TypePassword:
		return huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(f.Label).
					Description(f.Description).
					EchoMode(huh.EchoModePassword).
					Value(&strVal),
			),
		), &strVal, nil

	default:
		strVal = current
		input := huh.NewInput().
			Title(f.Label).
			Description(f.Description).
			Value(&strVal)
		if f.Validate != nil {
			input = input.Validate(f.Validate)
		}
		return huh.NewForm(
			huh.NewGroup(input),
		), &strVal, nil
	}
}
