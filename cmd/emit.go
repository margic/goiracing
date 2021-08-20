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

// emitCmd represents the emit command
var emitCmd = &cobra.Command{
	Use:   "emit",
	Short: "Emits iRacing Telemetry Data",
	Long: `Emitter for iRacing telemetry. Sets up goiracing to Output options
		can be modified with flags see goiracing emit --help for details
		The intention of emit is to enalbe goiracing to continually read `,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := &iracing.ClientConfig{
			Debug: true,
		}

		client := iracing.NewClient(cfg)

		client.Open()
		defer client.Close()
	},
}

func init() {
	rootCmd.AddCommand(emitCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// emitCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// emitCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
