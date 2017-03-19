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

	"log"

	"github.com/spf13/cobra"
	mgo "gopkg.in/mgo.v2"
)

type dataConfigCMD struct {
	*cobra.Command
}

// dataconfigCmd represents the dataconfig command
var dataconfigCmd = &cobra.Command{
	Use:   "dataconfig",
	Short: "data config configures the data service",
	Long:  `data config connects to a given environment data service and creates a new database and user name password combo based on the arguments passed in.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, err := cmd.Flags().GetString("dbhost")
		if err != nil {
			log.Fatalf("failed to get flag --dbhost %s", err.Error())
		}
		dbAdminPass, err := cmd.Flags().GetString("admin-pass")
		if err != nil {
			log.Fatalf("failed to get flag --admin-pass %s", err.Error())
		}
		replSet, err := cmd.Flags().GetString("replicaSet")
		if err != nil {
			log.Fatalf("failed to get flag --admin-pass %s", err.Error())
		}
		mongoURL := fmt.Sprintf("mongodb://admin:%s@%s/admin", dbAdminPass, host)
		if replSet != "" {
			mongoURL += "?replicaSet=" + replSet
		}
		database, err := cmd.Flags().GetString("database")
		if err != nil {
			log.Fatalf("failed to get flag --database %s", err.Error())
		}
		databaseUser, err := cmd.Flags().GetString("database-user")
		if err != nil {
			log.Fatalf("failed to get flag --database-user %s", err.Error())
		}
		databasePass, err := cmd.Flags().GetString("database-pass")
		if err != nil {
			log.Fatalf("failed to get flag --database-pass %s", err.Error())
		}
		session, err := mgo.Dial(mongoURL)
		if err != nil {
			log.Fatalf("failed to connect to mongo %s", err.Error())
		}
		defer session.Close()

		if err := session.DB(database).UpsertUser(&mgo.User{
			Username: databaseUser,
			Password: databasePass,
			Roles:    []mgo.Role{mgo.RoleReadWrite},
		}); err != nil {
			log.Fatalf("failed to create database user %s", err.Error())
		}
	},
}

func init() {
	RootCmd.AddCommand(dataconfigCmd)
	dataconfigCmd.Flags().StringP("dbhost", "d", "", "db host")
	dataconfigCmd.Flags().StringP("admin-user", "a", "", "admin user name")
	dataconfigCmd.Flags().StringP("admin-pass", "p", "", "admin pasword")
	dataconfigCmd.Flags().String("database", "db", "database to create")
	dataconfigCmd.Flags().String("database-user", "", "database  user to create")
	dataconfigCmd.Flags().String("database-pass", "", "database  pass to create")
	dataconfigCmd.Flags().String("replicaSet", "", "replicaSet to use")

}
