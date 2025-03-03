package api_test

import (
	"testing"

	"github.com/pluralsh/gqlclient"
	"github.com/pluralsh/plural/pkg/api"

	"github.com/stretchr/testify/assert"
)

func TestConstructGqlClientRepositoryInput(t *testing.T) {
	devopsCategory := gqlclient.CategoryDevops
	testDescription := "test"
	privateFlag := true
	icon := "plural/icons/test.png"
	notes := "plural/notes.tpl"
	name := "test"
	emptyString := ""

	tests := []struct {
		name     string
		input    string
		expected *gqlclient.RepositoryAttributes
	}{
		{
			name: `test repository.yaml conversion`,
			expected: &gqlclient.RepositoryAttributes{
				Category:    &devopsCategory,
				DarkIcon:    &emptyString,
				Description: &testDescription,
				GitURL:      &emptyString,
				Homepage:    &emptyString,
				Icon:        &icon,
				Name:        &name,
				Notes:       &notes,
				OauthSettings: &gqlclient.OauthSettingsAttributes{
					AuthMethod: "POST",
					URIFormat:  "https://{domain}/oauth2/callback",
				},
				Private: &privateFlag,
				Readme:  nil,
				Secrets: nil,
				Tags: []*gqlclient.TagAttributes{
					{
						Tag: "data-science",
					},
				},
				Verified: nil,
			},
			input: `name: test
description: test
category: DEVOPS
private: true
icon: plural/icons/test.png
notes: plural/notes.tpl
oauthSettings:
  uriFormat: https://{domain}/oauth2/callback
  authMethod: POST
tags:
- tag: data-science
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repositoryAttributes, err := api.ConstructGqlClientRepositoryInput([]byte(test.input))
			assert.NoError(t, err)
			assert.Equal(t, repositoryAttributes, test.expected)
		})
	}
}
