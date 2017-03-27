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
	"database/sql"
	"log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	_ "github.com/go-sql-driver/mysql" //mysql driver used by sql package
)

// dataMysqlConfigCmd represents the dataMysqlConfig command
var dataMysqlConfigCmd = &cobra.Command{
	Use:     "datamysqlconfig",
	Short:   "Configure MySql with a new user/pass/db",
	Long:    `This tool will create a new user connected to a new database in an existing mysql service.`,
	PreRunE: prerunCheck,
	Run:     configMysql,
}

func init() {
	RootCmd.AddCommand(dataMysqlConfigCmd)
	dataMysqlConfigCmd.Flags().StringP("host", "", "", "db host")
	dataMysqlConfigCmd.Flags().StringP("admin-username", "", "", "admin user name")
	dataMysqlConfigCmd.Flags().StringP("admin-password", "", "", "admin pasword")
	dataMysqlConfigCmd.Flags().String("admin-database", "", "admin database")
	dataMysqlConfigCmd.Flags().String("user-username", "", "database  user to create")
	dataMysqlConfigCmd.Flags().String("user-password", "", "database  pass to create")
	dataMysqlConfigCmd.Flags().String("user-database", "", "database  pass to create")

}

func getArgValue(cmd *cobra.Command, arg string) (string, error) {
	var err error
	value, err := cmd.Flags().GetString(arg)
	if len(value) == 0 {
		err = errors.New("No value provided for argument")
	}
	return value, err
}

func prerunCheck(cmd *cobra.Command, args []string) error {
	var err error
	_, err = getArgValue(cmd, "host")
	if err != nil {
		return errors.Wrap(err, "failed to get flag --host")
	}
	_, err = getArgValue(cmd, "admin-username")
	if err != nil {
		return errors.Wrap(err, "failed to get flag --admin-username")
	}
	_, err = getArgValue(cmd, "admin-password")
	if err != nil {
		return errors.Wrap(err, "failed to get flag --admin-password")
	}
	_, err = getArgValue(cmd, "admin-database")
	if err != nil {
		return errors.Wrap(err, "failed to get flag --admin-database")
	}
	_, err = getArgValue(cmd, "user-username")
	if err != nil {
		return errors.Wrap(err, "failed to get flag --user-username")
	}
	_, err = getArgValue(cmd, "user-password")
	if err != nil {
		return errors.Wrap(err, "failed to get flag --user-password")
	}
	_, err = getArgValue(cmd, "user-database")
	if err != nil {
		return errors.Wrap(err, "failed to get flag --user-database")
	}

	return nil
}

func configMysql(cmd *cobra.Command, args []string) {
	//error checking already completed in preRunE
	host, _ := getArgValue(cmd, "host")
	adminUser, _ := getArgValue(cmd, "admin-username")
	adminPass, _ := getArgValue(cmd, "admin-password")
	adminDb, _ := getArgValue(cmd, "admin-database")
	username, _ := getArgValue(cmd, "user-username")
	userPass, _ := getArgValue(cmd, "user-password")
	userDb, _ := getArgValue(cmd, "user-database")

	dsn := adminUser + ":" + adminPass + "@tcp(" + host + ":3306)/" + adminDb
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to parse DSN for MySQL Connection ("+dsn+")", err.Error())
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("failed to connect to MySQL with DSN ("+dsn+")", err.Error())
	}

	stmtCreateDatabase, err := db.Prepare("CREATE DATABASE IF NOT EXISTS " + userDb)
	if err != nil {
		log.Fatalf("failed to prepare the create database statement: %s", err.Error())
	}
	defer stmtCreateDatabase.Close()
	if _, err := stmtCreateDatabase.Exec(); err != nil {
		log.Fatalf("error executing create database query: %s", err.Error())
	}

	//Grant will create the user if missing, and update the password if altered.
	stmtGrantUserPerms, err := db.Prepare("GRANT ALL PRIVILEGES ON " + userDb + ".* TO '" + username + "'@'%' IDENTIFIED BY '" + userPass + "';")
	if err != nil {
		log.Fatalf("failed to prepare the grant user permissions statement: %s", err.Error())
	}
	defer stmtGrantUserPerms.Close()

	if _, err := stmtGrantUserPerms.Exec(); err != nil {
		log.Fatalf("error executing grant user permissions query: %s", err.Error())
	}
}
