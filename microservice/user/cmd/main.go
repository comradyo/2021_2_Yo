package main

import (
	proto "backend/microservice/user/proto"
	"backend/microservice/user/repository"
	log "backend/pkg/logger"
	"backend/pkg/utils"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"net"
	"os"
)

const logMessage = "microservice:event:"


func main() {

	logLevel := logrus.DebugLevel
	log.Init(logLevel)

	log.Info(logMessage + "started")

	viper.AddConfigPath("../config")
	viper.SetConfigName("config")
	err := viper.ReadInConfig()
	if err != nil {
		log.Error(logMessage+"err =", err)
		os.Exit(1)
	}

	db, err := utils.InitPostgresDB()
	if err != nil {
		log.Error(logMessage+"err =", err)
		os.Exit(1)
	}

	port := viper.GetString("user_port")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Error(logMessage+"err =", err)
		os.Exit(1)
	}

	server := grpc.NewServer()

	userRepositoryService := repository.NewRepository(db)
	proto.RegisterRepositoryServer(server, userRepositoryService)

	log.Info("started user microservice on ", port)
	err = server.Serve(listener)
	if err != nil {
		log.Error(logMessage+"err =", err)
		os.Exit(1)
	}

}
