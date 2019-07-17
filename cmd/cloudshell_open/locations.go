package main

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

func listLocations() ([]string, error) {
	loc := []string{
		`us-central1`,
		`asia-northeast1`,
		`euro-west1`,
		`us-east1`,
	}
	return loc, nil
}

func promptLocation(locations []string) (string, error) {
	var p string
	if err := survey.AskOne(&survey.Select{
		Message: "Choose a region to deploy this application:",
		Options: locations,
	}, &p,
		surveyIconOpts,
		survey.WithValidator(survey.Required),
	); err != nil {
		return p, fmt.Errorf("could not choose a region: %+v", err)
	}
	return p, nil

}
