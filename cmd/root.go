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

var (
	destination string
	concurrency uint
)

var rootCmd = &cobra.Command{
	Use:   "8gooses <URL / Partial URL>",
	Short: "8gooses",
	Args:  cobra.MinimumNArgs(1),
	Long: `
8gooses: An 8muses comic downloader in Go
`,
	Run: func(cmd *cobra.Command, args []string) {
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
		download.Tasks.Wait()
	},
}

// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// when this action is called directly.
	rootCmd.Flags().StringVarP(&destination, "output", "o", ".", "Directory to save the downloaded comics.")
	rootCmd.Flags().UintVarP(&concurrency, "concurrency", "c", 4, "Number of coroutines to use when downloading.")
}
