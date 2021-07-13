/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/margic/goiracing/iracing"
	"github.com/spf13/cobra"
)

// sessionCmd represents the session command
var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Ouput iRacing Session Information",
	Long: `Output iRacing Session Infomraiton. Dumps the yaml formatted string of
		session information from iRacing. Use flags to direct output as required.`,
	Run: func(cmd *cobra.Command, args []string) {
		sessioninfo()
	},
}

func sessioninfo() {
	client := iracing.NewClient()
	client.Open()
	defer client.Close()
}

func init() {
	rootCmd.AddCommand(sessionCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sessionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sessionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
