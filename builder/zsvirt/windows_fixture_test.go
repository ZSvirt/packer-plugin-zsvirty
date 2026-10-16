// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zsvirt

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// WinPE locale settings alone leave Windows 11 waiting at the OOBE region
// and keyboard pages, preventing FirstLogonCommands from enabling WinRM.
func TestWindows11AnswerFileConfiguresOOBELocale(t *testing.T) {
	data, err := os.ReadFile("../../examples/iso/windows11-Autounattend.xml.pkrtpl")
	require.NoError(t, err)
	var answer struct {
		Settings []struct {
			Pass       string `xml:"pass,attr"`
			Components []struct {
				Name         string `xml:"name,attr"`
				InputLocale  string
				SystemLocale string
				UILanguage   string
				UserLocale   string
			} `xml:"component"`
		} `xml:"settings"`
	}
	require.NoError(t, xml.Unmarshal(data, &answer))
	for _, settings := range answer.Settings {
		if settings.Pass != "oobeSystem" {
			continue
		}
		for _, component := range settings.Components {
			if component.Name == "Microsoft-Windows-International-Core" {
				require.NotEmpty(t, component.InputLocale)
				require.NotEmpty(t, component.SystemLocale)
				require.NotEmpty(t, component.UILanguage)
				require.NotEmpty(t, component.UserLocale)
				return
			}
		}
	}
	t.Fatal("Windows 11 answer file must configure the oobeSystem locale component")
}
