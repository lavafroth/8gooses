/*
Copyright © 2022 Himadri Bhattacharjee

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"log"
	"os"

	"github.com/lavafroth/8gooses/pkg/download"
	"github.com/lavafroth/8gooses/pkg/resource"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "8gooses <URL / Partial URL>",
	Short: "8gooses",
	Long:  `
8gooses: An 8muses comic downloader in Go
`,
	Run: func(cmd *cobra.Command, args []string) {
		destination, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatalln(err)
		}
		concurrency, err := cmd.Flags().GetUint("concurrency")
		if err != nil {
			log.Fatalln(err)
		}
		download.StartJobs(concurrency)
		for _, arg := range args {
			tags := resource.Tags(arg)

			// Default to downloading a single episode
			action := download.EPISODE
			switch len(tags) {
			case 1:
				// Download all episodes by an artist
				action = download.ARTIST
			case 2:
				// Download all episodes in the album
				action = download.ALBUM
			}
			if err := download.Traverse(tags, destination, action); err != nil {
				log.Fatalln(err)
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.8gooses.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringP("output", "o", ".", "Directory to save the downloaded comics.")
	rootCmd.Flags().UintP("concurrency", "c", 4, "Number of coroutines to use when downloading.")
}
