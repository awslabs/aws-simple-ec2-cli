// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package cmd

import (
	"fmt"
	"os"
	"simple-ec2/pkg/version"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "simple-ec2",
	Short: "AWS Simple EC2 CLI (simple-ec2) is a simple tool to launch, connect and terminate Amazon EC2 instances",
	Long: "AWS Simple EC2 CLI (simple-ec2) is a simple tool to launch, connect and terminate Amazon EC2 instances. " +
		"Users can easily launch an instance with or without custom configurations.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of this binary",
	Long:  `Print the version number of this binary`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version %s", version.BuildInfo)
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
