// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"os"

	"github.com/feedhenry/negotiator/deploy"
	"github.com/feedhenry/negotiator/pkg/openshift"
	"github.com/spf13/cobra"
)

var (
	repoLoc, repRef, env, serviceName string
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploys a new service to an OSCP project",
	Long: `deploy <serviceName> --token=<your_oscp_token> --host=<an oscp master url> --name=<someName> --env=environment :
	
	flags required when deploying a cloudapp --repo="https://github.com/feedhenry/testing-cloud-app.git" --repo-ref=master`,
	Run: func(cmd *cobra.Command, args []string) {
		tl := openshift.NewTemplateLoaderDecoder("")
		if len(args) != 1 {
			fmt.Println("error: not enough args\n", cmd.Long)
			return
		}
		template := args[0]
		deployController := deploy.New(tl, tl)
		payload := deploy.Payload{
			Repo: &deploy.Repo{
				Loc: repoLoc,
				Ref: repRef,
			},
			ServiceName: serviceName,
			Replicas:    1,
		}
		clientFactory := openshift.ClientFactory{}
		client, err := clientFactory.DefaultDeployClient(host, token)
		if err != nil {
			fmt.Printf("error: failed to deploy template %s ", err.Error())
			os.Exit(-1)
		}
		if err := deployController.Template(client, template, env, &payload); err != nil {
			fmt.Printf("error: failed to deploy template %s ", err.Error())
			os.Exit(-1)
		}
	},
}

func init() {
	deployCmd.Flags().StringVar(&repoLoc, "repo", "", "--repo=https://github.com/feedhenry/testing-cloud-app.git")
	deployCmd.Flags().StringVar(&repRef, "repo-ref", "", "--repo-ref=master")
	deployCmd.Flags().StringVar(&env, "env", "", "--env=environment")
	deployCmd.Flags().StringVar(&serviceName, "name", "", "--name=servicename")
	RootCmd.AddCommand(deployCmd)

}
