package main

import (
	"errors"
	"os"

	"golang.org/x/crypto/bcrypt"

	"github.com/codegangsta/cli"
	"github.com/jinzhu/gorm"
	"github.com/quorumsco/cmd"
	"github.com/quorumsco/databases"
	"github.com/quorumsco/logs"
	"github.com/quorumsco/oauth2/models"
	"github.com/quorumsco/settings"
)

func main() {
	cmd := cmd.New()
	cmd.Name = "addusers"
	cmd.Usage = "add users to quorums' database"
	cmd.Version = "0.0.1"
	cmd.Before = add
	cmd.Flags = append(cmd.Flags, []cli.Flag{
		cli.StringFlag{Name: "config, c", Usage: "configuration file", EnvVar: "CONFIG"},
		cli.StringFlag{Name: "mail, m", Usage: "email"},
		cli.StringFlag{Name: "password, p", Usage: "password"},
		cli.StringFlag{Name: "firstname, f", Usage: "firstname"},
		cli.StringFlag{Name: "surname, s", Usage: "surname"},
		cli.HelpFlag,
	}...)
	cmd.RunAndExitOnError()
}

func sPtr(s string) *string {
	if s == "" {
		return nil
	} else {
		return &s
	}
}

func add(ctx *cli.Context) error {
	var err error
	var config settings.Config
	if ctx.String("config") != "" {
		config, err = settings.Parse(ctx.String("config"))
		if err != nil {
			logs.Error(err)
		}
	}

	var mail = ctx.String("mail")
	var password = ctx.String("password")
	var firstname = ctx.String("firstname")
	var surname = ctx.String("surname")

	if mail == "" || password == "" || firstname == "" || surname == "" {
		logs.Error("All arguments are required")
		return errors.New("all arguments are required")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	u := &models.User{
		Mail:      sPtr(mail),
		Password:  sPtr(string(passwordHash)),
		Firstname: sPtr(firstname),
		Surname:   sPtr(surname),
	}
	errs := u.Validate()

	logs.Level(logs.DebugLevel)

	if len(errs) > 0 {
		logs.Error(errs)
		return errors.New("Informations are not valid")
	}

	dialect, args, err := config.SqlDB()
	if err != nil {
		logs.Critical(err)
		os.Exit(1)
	}
	logs.Debug("database type: %s", dialect)

	var db *gorm.DB
	if db, err = databases.InitGORM(dialect, args); err != nil {
		logs.Critical(err)
		os.Exit(1)
	}
	logs.Debug("connected to %s", args)

	if config.Migrate() {
		db.AutoMigrate(models.Models()...)
		logs.Debug("database migrated successfully")
	}

	db.LogMode(true)

	var store = models.UserStore(db)
	err = store.Save(u)
	if err != nil {
		logs.Error(err)
		return err
	}

	logs.Debug("New user :  -Mail : %s  -Password : %s", mail, password)
	return nil
}
